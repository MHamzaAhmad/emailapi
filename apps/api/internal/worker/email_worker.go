package worker

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/jordan-wright/email"
	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
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
	sesClient ses.Client
	s3Factory *s3.Factory
	chRepo    chrepo.EmailRepositoryInterface
}

// NewEmailWorker creates a new EmailWorker.
func NewEmailWorker(
	sesClient ses.Client,
	s3Factory *s3.Factory,
	chRepo chrepo.EmailRepositoryInterface,
) *EmailWorker {
	return &EmailWorker{
		sesClient: sesClient,
		s3Factory: s3Factory,
		chRepo:    chRepo,
	}
}

func (w *EmailWorker) Work(ctx context.Context, job *river.Job[SendEmailArgs]) error {
	args := job.Args

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
	if err := w.chRepo.InsertRouting(ctx, messageID, args.EmailID, args.UserID); err != nil {
		fmt.Printf("Warning: failed to insert routing entry: %v\n", err)
	}

	// Log success
	w.logActivity(ctx, args, "sent", fmt.Sprintf("Message ID: %s", messageID))

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

	if w.chRepo != nil {
		w.chRepo.LogActivity(ctx, args.UserID, "email", args.EmailID, action, status, details, metadata)
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

	for _, att := range attachments {
		_, err := e.Attach(bytes.NewReader(att.Data), att.Filename, att.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to attach file %s: %w", att.Filename, err)
		}
	}

	return e.Bytes()
}

// ScheduledEmailArgs is for scheduled emails (processed at scheduled_at time).
type ScheduledEmailArgs struct {
	SendEmailArgs
	ScheduledAt time.Time `json:"scheduled_at"`
}

func (ScheduledEmailArgs) Kind() string { return "scheduled_email" }
