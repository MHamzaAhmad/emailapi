package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/jordan-wright/email"
	"github.com/riverqueue/river"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/external/polar"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
	"github.com/emailapi/api/internal/webhook"
)

// SendEmailArgs contains all email data embedded in the job payload.
// This is the transient storage - data is cleaned up when job completes.
type SendEmailArgs struct {
	EmailID    string            `json:"email_id"`
	UserID     string            `json:"user_id"`
	From       string            `json:"from"`
	To         []string          `json:"to"`
	Cc         []string          `json:"cc,omitempty"`
	Bcc        []string          `json:"bcc,omitempty"`
	Subject    string            `json:"subject"`
	Body       string            `json:"body,omitempty"`
	HTML       string            `json:"html,omitempty"`
	InReplyTo  string            `json:"in_reply_to,omitempty"`
	References []string          `json:"references,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	// S3 keys of attachments (populated by attachment worker)
	AttachmentKeys []AttachmentInfo `json:"attachment_keys,omitempty"`
	// Optional scheduled time for email delivery
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	// DryRun mode skips SES and returns fake message ID (for performance testing)
	DryRun bool `json:"dry_run,omitempty"`
	// UnsubscribeURL for List-Unsubscribe header (RFC 2369 + RFC 8058)
	UnsubscribeURL string `json:"unsubscribe_url,omitempty"`
	// Unsubscribe config for generating per-recipient links (set when placeholder present)
	UnsubscribeBaseURL     string `json:"unsubscribe_base_url,omitempty"`
	UnsubscribeTokenSecret string `json:"unsubscribe_token_secret,omitempty"`
}

// AttachmentInfo contains info about an attachment stored in S3.
type AttachmentInfo struct {
	S3Key       string `json:"s3_key"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

func (SendEmailArgs) Kind() string { return "send_email" }

// EmailWorker handles email sending jobs.
type EmailWorker struct {
	river.WorkerDefaults[SendEmailArgs]
	sesClient     ses.Client
	s3Factory     s3.FactoryInterface
	tbRepo        tbrepo.EmailRepositoryInterface
	webhookSender webhook.Sender
	// Usage tracking
	creditCache redisrepo.CreditCacheInterface
	polarClient polar.Client
}

// NewEmailWorker creates a new EmailWorker.
func NewEmailWorker(
	sesClient ses.Client,
	s3Factory s3.FactoryInterface,
	tbRepo tbrepo.EmailRepositoryInterface,
	webhookSender webhook.Sender,
	creditCache redisrepo.CreditCacheInterface,
	polarClient polar.Client,
) *EmailWorker {
	return &EmailWorker{
		sesClient:     sesClient,
		s3Factory:     s3Factory,
		tbRepo:        tbRepo,
		webhookSender: webhookSender,
		creditCache:   creditCache,
		polarClient:   polarClient,
	}
}

func (w *EmailWorker) Work(ctx context.Context, job *river.Job[SendEmailArgs]) error {
	args := job.Args

	// Handle dry-run mode: skip SES, simulate delay, return fake message ID
	if args.DryRun {
		// Simulate realistic SES latency (50-150ms)
		delay := 50*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
		time.Sleep(delay)

		fakeMessageID := fmt.Sprintf("dry-run-%s@simpleemailapi.dev", args.EmailID[:8])
		fmt.Printf("[DRY-RUN] Simulated sending email %s to %v (MsgID: %s)\n", args.EmailID, args.To, fakeMessageID)
		return nil
	}

	// Check if we need to split for unsubscribe placeholders
	hasPlaceholder := strings.Contains(args.Body, "{{unsubscribe_link}}") || strings.Contains(args.HTML, "{{unsubscribe_link}}")
	totalRecipients := len(args.To) + len(args.Cc) + len(args.Bcc)

	if hasPlaceholder && totalRecipients > 1 && args.UnsubscribeTokenSecret != "" {
		// Split into individual sends for proper unsubscribe tracking
		return w.sendSplitEmails(ctx, &args)
	}

	// Download attachments from S3 in parallel
	attachmentData := make([]attachmentContent, len(args.AttachmentKeys))
	downloadErrs := make([]error, len(args.AttachmentKeys))

	var wg sync.WaitGroup
	for i, att := range args.AttachmentKeys {
		wg.Add(1)
		go func(idx int, att AttachmentInfo) {
			defer wg.Done()
			data, err := w.s3Factory.Bucket(s3.BucketAttachments).Download(ctx, att.S3Key)
			if err != nil {
				downloadErrs[idx] = fmt.Errorf("failed to download attachment %s: %w", att.Filename, err)
				return
			}
			attachmentData[idx] = attachmentContent{
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Data:        data,
			}
		}(i, att)
	}
	wg.Wait()

	// Check for download errors
	for _, err := range downloadErrs {
		if err != nil {
			w.logActivity(ctx, args, "failed", fmt.Sprintf("Failed to download attachment: %v", err))
			return err
		}
	}

	// Send via SES
	messageID, err := w.sendEmail(ctx, &args, attachmentData)
	if err != nil {
		w.logActivity(ctx, args, "failed", err.Error())
		return fmt.Errorf("failed to send email: %w", err)
	}

	// Write routing entry for reply tracking
	if err := w.tbRepo.InsertRouting(ctx, messageID, args.EmailID, args.UserID); err != nil {
		fmt.Printf("Warning: failed to insert routing entry: %v\n", err)
	}

	// Log success
	w.logActivity(ctx, args, "sent", fmt.Sprintf("Message ID: %s", messageID))

	// Track usage after successful send
	w.trackUsage(ctx, args.UserID, 1)

	// Send webhook notification
	w.sendWebhook(ctx, args, messageID)

	// Clean up attachments from S3 after successful send (fire-and-forget, parallel)
	for _, att := range args.AttachmentKeys {
		go func(key string) {
			if err := w.s3Factory.Bucket(s3.BucketAttachments).DeleteObject(context.Background(), key); err != nil {
				fmt.Printf("Warning: failed to delete attachment %s from S3: %v\n", key, err)
			}
		}(att.S3Key)
	}

	fmt.Printf("Sent email %s to %v (MsgID: %s)\n", args.EmailID, args.To, messageID)
	return nil
}

func (w *EmailWorker) logActivity(ctx context.Context, args SendEmailArgs, action, details string) {
	status := "success"
	if action == "failed" {
		status = "failed"
	}

	metadata := map[string]interface{}{
		"from":            args.From,
		"to":              args.To,
		"subject":         args.Subject,
		"has_attachments": len(args.AttachmentKeys) > 0,
	}
	if args.Metadata != nil {
		for k, v := range args.Metadata {
			metadata[k] = v
		}
	}

	if w.tbRepo != nil {
		w.tbRepo.LogActivity(ctx, args.UserID, "email", args.EmailID, action, status, details, metadata)
	}
}

type attachmentContent struct {
	Filename    string
	ContentType string
	Data        []byte
}

// sendEmail sends the email via SES using raw MIME message.
func (w *EmailWorker) sendEmail(ctx context.Context, args *SendEmailArgs, attachments []attachmentContent) (string, error) {
	rawMessage, err := buildMIMEMessage(args, attachments)
	if err != nil {
		return "", fmt.Errorf("failed to build MIME message: %w", err)
	}

	input := &sesv2.SendEmailInput{
		Content: &types.EmailContent{
			Raw: &types.RawMessage{
				Data: rawMessage,
			},
		},
	}

	resp, err := w.sesClient.SendEmail(ctx, input)
	if err != nil {
		return "", err
	}

	return aws.ToString(resp.MessageId), nil
}

// buildMIMEMessage constructs a MIME message using jordan-wright/email library.
func buildMIMEMessage(args *SendEmailArgs, attachments []attachmentContent) ([]byte, error) {
	e := email.NewEmail()

	e.From = args.From
	e.To = args.To
	e.Cc = args.Cc
	e.Bcc = args.Bcc
	e.Subject = args.Subject

	if args.Body != "" {
		e.Text = []byte(args.Body)
	}
	if args.HTML != "" {
		e.HTML = []byte(args.HTML)
	}

	// Add In-Reply-To header if present
	if args.InReplyTo != "" {
		e.Headers.Add("In-Reply-To", args.InReplyTo)
	}

	// Add References header if present (for proper threading)
	if len(args.References) > 0 {
		e.Headers.Add("References", strings.Join(args.References, " "))
	}

	// Add List-Unsubscribe headers (RFC 2369 + RFC 8058)
	if args.UnsubscribeURL != "" {
		// RFC 2369 - List-Unsubscribe header with URL
		e.Headers.Add("List-Unsubscribe", fmt.Sprintf("<%s>", args.UnsubscribeURL))
		// RFC 8058 - One-click unsubscribe support
		e.Headers.Add("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
	}

	for _, att := range attachments {
		_, err := e.Attach(bytes.NewReader(att.Data), att.Filename, att.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to attach file %s: %w", att.Filename, err)
		}
	}

	return e.Bytes()
}

// sendWebhook sends a webhook notification for email events.
func (w *EmailWorker) sendWebhook(ctx context.Context, args SendEmailArgs, messageID string) {
	if w.webhookSender == nil {
		return
	}

	event := &v1.EmailSentEvent{
		EmailId:   args.EmailID,
		UserId:    args.UserID,
		MessageId: messageID,
		From:      args.From,
		To:        args.To,
		Cc:        args.Cc,
		Bcc:       args.Bcc,
		Subject:   args.Subject,
		Metadata:  args.Metadata,
	}

	w.webhookSender.SendEmailSent(ctx, args.UserID, event)
}

// trackUsage increments usage counters and ingests billing event after successful send.
func (w *EmailWorker) trackUsage(ctx context.Context, userID string, count int64) {
	// Increment daily usage counter (for limit checking)
	if w.creditCache != nil {
		if err := w.creditCache.IncrDailyUsage(ctx, userID, count); err != nil {
			fmt.Printf("Warning: failed to increment daily usage for %s: %v\n", userID, err)
		}
		// Increment consumed counter (for monthly tracking)
		if err := w.creditCache.IncrConsumed(ctx, userID, count); err != nil {
			fmt.Printf("Warning: failed to increment consumed for %s: %v\n", userID, err)
		}
	}

	// Ingest event to Polar for billing
	if w.polarClient != nil {
		if err := w.polarClient.IngestEmailEvent(ctx, userID, count); err != nil {
			fmt.Printf("Warning: failed to ingest Polar event for %s: %v\n", userID, err)
		}
	}
}

// sendSplitEmails splits a multi-recipient email into individual sends.
// Each recipient gets their own unique unsubscribe link.
// NOTE: Each individual email counts as a separate send.
func (w *EmailWorker) sendSplitEmails(ctx context.Context, args *SendEmailArgs) error {
	// Collect all recipients
	var allRecipients []string
	allRecipients = append(allRecipients, args.To...)
	allRecipients = append(allRecipients, args.Cc...)
	allRecipients = append(allRecipients, args.Bcc...)

	fmt.Printf("Splitting email %s into %d individual sends for unsubscribe tracking\n", args.EmailID, len(allRecipients))

	// Download attachments once (shared across all sends)
	attachmentData := make([]attachmentContent, len(args.AttachmentKeys))
	for i, att := range args.AttachmentKeys {
		data, err := w.s3Factory.Bucket(s3.BucketAttachments).Download(ctx, att.S3Key)
		if err != nil {
			return fmt.Errorf("failed to download attachment %s: %w", att.Filename, err)
		}
		attachmentData[i] = attachmentContent{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Data:        data,
		}
	}

	// Send to each recipient individually
	var lastErr error
	successCount := 0

	for i, recipient := range allRecipients {
		// Generate unique unsubscribe link for this recipient
		unsubLink := w.generateUnsubscribeLink(args, recipient)

		// Clone args for this recipient
		individualArgs := *args
		individualArgs.To = []string{recipient}
		individualArgs.Cc = nil
		individualArgs.Bcc = nil
		individualArgs.UnsubscribeURL = unsubLink

		// Replace placeholder with this recipient's link
		if unsubLink != "" {
			individualArgs.Body = strings.ReplaceAll(args.Body, "{{unsubscribe_link}}", unsubLink)
			individualArgs.HTML = strings.ReplaceAll(args.HTML, "{{unsubscribe_link}}", unsubLink)
		}

		// Send this individual email
		messageID, err := w.sendEmail(ctx, &individualArgs, attachmentData)
		if err != nil {
			lastErr = err
			w.logActivity(ctx, individualArgs, "failed", err.Error())
			fmt.Printf("  [%d/%d] Failed to send to %s: %v\n", i+1, len(allRecipients), recipient, err)
			continue
		}

		successCount++

		// Log and webhook for each individual send
		w.logActivity(ctx, individualArgs, "sent", fmt.Sprintf("Message ID: %s (split %d/%d)", messageID, i+1, len(allRecipients)))
		w.trackUsage(ctx, args.UserID, 1) // Each split counts as 1
		w.sendWebhook(ctx, individualArgs, messageID)

		// Insert routing entry
		if w.tbRepo != nil {
			w.tbRepo.InsertRouting(ctx, messageID, args.EmailID, args.UserID)
		}

		fmt.Printf("  [%d/%d] Sent to %s (MsgID: %s)\n", i+1, len(allRecipients), recipient, messageID)
	}

	// Clean up attachments after all sends
	for _, att := range args.AttachmentKeys {
		go func(key string) {
			w.s3Factory.Bucket(s3.BucketAttachments).DeleteObject(context.Background(), key)
		}(att.S3Key)
	}

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("all %d sends failed, last error: %w", len(allRecipients), lastErr)
	}

	fmt.Printf("Split email %s: %d/%d successful\n", args.EmailID, successCount, len(allRecipients))
	return nil
}

// generateUnsubscribeLink creates an unsubscribe link for a specific recipient.
func (w *EmailWorker) generateUnsubscribeLink(args *SendEmailArgs, recipient string) string {
	if args.UnsubscribeTokenSecret == "" || args.UnsubscribeBaseURL == "" {
		return ""
	}

	// Create token data
	tokenData := map[string]interface{}{
		"u": args.UserID,
		"e": recipient,
		"i": args.EmailID,
		"x": time.Now().Add(30 * 24 * time.Hour).Unix(),
	}

	payload, err := json.Marshal(tokenData)
	if err != nil {
		return ""
	}

	// Encode payload
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	// Compute HMAC signature
	h := hmac.New(sha256.New, []byte(args.UnsubscribeTokenSecret))
	h.Write(payload)
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	// Combine token
	token := encodedPayload + "." + signature

	return fmt.Sprintf("%s/unsubscribe?token=%s", strings.TrimSuffix(args.UnsubscribeBaseURL, "/"), token)
}
