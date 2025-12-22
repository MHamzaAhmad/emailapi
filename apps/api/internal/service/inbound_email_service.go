package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/svix"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	pgrepo "github.com/emailapi/api/internal/repository/postgres"
)

// InboundEmailService handles inbound email processing (replies) from SNS/SES.
type InboundEmailService struct {
	s3Client      s3.Client
	pgRepo        *pgrepo.EmailRepository
	chRepo        *chrepo.EmailRepository
	svixClient    svix.Client
	inboundBucket string
}

// NewInboundEmailService creates a new InboundEmailService.
func NewInboundEmailService(
	s3Client s3.Client,
	pgRepo *pgrepo.EmailRepository,
	chRepo *chrepo.EmailRepository,
	svixClient svix.Client,
	inboundBucket string,
) *InboundEmailService {
	return &InboundEmailService{
		s3Client:      s3Client,
		pgRepo:        pgRepo,
		chRepo:        chRepo,
		svixClient:    svixClient,
		inboundBucket: inboundBucket,
	}
}

// SNSNotification represents the parsed SNS notification payload.
type SNSNotification struct {
	Type             string `json:"Type"`
	MessageID        string `json:"MessageId"`
	TopicArn         string `json:"TopicArn"`
	Message          string `json:"Message"`
	SubscribeURL     string `json:"SubscribeURL"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
}

// SESNotification represents the SES notification inside SNS Message.
type SESNotification struct {
	NotificationType string `json:"notificationType"`
	Receipt          struct {
		Action struct {
			Type       string `json:"type"`
			BucketName string `json:"bucketName"`
			ObjectKey  string `json:"objectKey"`
		} `json:"action"`
	} `json:"receipt"`
	Mail struct {
		MessageID   string   `json:"messageId"`
		Source      string   `json:"source"`
		Destination []string `json:"destination"`
	} `json:"mail"`
}

// InboundEmail represents a parsed inbound email.
type InboundEmail struct {
	ID              string   `json:"id"`
	MessageID       string   `json:"message_id"`
	InReplyTo       string   `json:"in_reply_to"`
	References      []string `json:"references"`
	From            string   `json:"from"`
	To              []string `json:"to"`
	Subject         string   `json:"subject"`
	Body            string   `json:"body"`
	HTML            string   `json:"html"`
	OriginalEmailID string   `json:"original_email_id"`
	UserID          string   `json:"user_id"`
}

// HandleSNSNotification processes SNS notifications from SES.
func (s *InboundEmailService) HandleSNSNotification(ctx context.Context, snsType, message, subscribeURL string) error {
	switch snsType {
	case "SubscriptionConfirmation":
		return s.handleSubscriptionConfirmation(ctx, subscribeURL)
	case "Notification":
		return s.handleNotification(ctx, message)
	case "UnsubscribeConfirmation":
		// Just log and acknowledge
		return nil
	default:
		return fmt.Errorf("unknown SNS notification type: %s", snsType)
	}
}

// handleSubscriptionConfirmation auto-confirms the SNS subscription.
func (s *InboundEmailService) handleSubscriptionConfirmation(ctx context.Context, subscribeURL string) error {
	if subscribeURL == "" {
		return fmt.Errorf("missing subscribe URL")
	}

	// Make a GET request to confirm the subscription
	resp, err := http.Get(subscribeURL)
	if err != nil {
		return fmt.Errorf("failed to confirm subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("subscription confirmation failed with status: %d", resp.StatusCode)
	}

	return nil
}

// handleNotification processes the actual email notification.
func (s *InboundEmailService) handleNotification(ctx context.Context, message string) error {
	// Parse SES notification from message
	var sesNotif SESNotification
	if err := json.Unmarshal([]byte(message), &sesNotif); err != nil {
		return fmt.Errorf("failed to parse SES notification: %w", err)
	}

	// Only process email received notifications with S3 action
	if sesNotif.NotificationType != "Received" {
		return nil // Ignore other notification types
	}

	if sesNotif.Receipt.Action.Type != "S3" {
		return fmt.Errorf("expected S3 action, got: %s", sesNotif.Receipt.Action.Type)
	}

	// Download raw email from S3
	bucket := sesNotif.Receipt.Action.BucketName
	key := sesNotif.Receipt.Action.ObjectKey

	// Use the configured inbound bucket if not specified in notification
	if bucket == "" {
		bucket = s.inboundBucket
	}

	rawEmail, err := s.s3Client.Download(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to download email from S3: %w", err)
	}

	// Parse the email
	inboundEmail, err := s.parseEmail(rawEmail)
	if err != nil {
		return fmt.Errorf("failed to parse email: %w", err)
	}

	// Look up original email by In-Reply-To header
	if inboundEmail.InReplyTo == "" {
		// No In-Reply-To header - check References
		if len(inboundEmail.References) == 0 {
			// Not a reply, skip
			return nil
		}
		// Use the last reference as the reply target
		inboundEmail.InReplyTo = inboundEmail.References[len(inboundEmail.References)-1]
	}

	// Find the original email to get the user ID
	originalEmail, err := s.lookupOriginalEmail(ctx, inboundEmail.InReplyTo)
	if err != nil {
		// Could be a reply to an email we didn't send
		return nil
	}

	inboundEmail.OriginalEmailID = originalEmail.ID
	inboundEmail.UserID = originalEmail.UserID

	// Deliver webhook to user via Svix
	return s.deliverWebhook(ctx, inboundEmail)
}

// parseEmail parses raw MIME email content.
func (s *InboundEmailService) parseEmail(rawEmail []byte) (*InboundEmail, error) {
	msg, err := mail.ReadMessage(strings.NewReader(string(rawEmail)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse email message: %w", err)
	}

	inbound := &InboundEmail{
		ID:        uuid.New().String(),
		MessageID: cleanMessageID(msg.Header.Get("Message-ID")),
		InReplyTo: cleanMessageID(msg.Header.Get("In-Reply-To")),
		From:      msg.Header.Get("From"),
		Subject:   msg.Header.Get("Subject"),
	}

	// Parse References header (space-separated message IDs)
	references := msg.Header.Get("References")
	if references != "" {
		refs := strings.Fields(references)
		for _, ref := range refs {
			inbound.References = append(inbound.References, cleanMessageID(ref))
		}
	}

	// Parse To addresses
	toHeader := msg.Header.Get("To")
	if toHeader != "" {
		addresses, err := mail.ParseAddressList(toHeader)
		if err == nil {
			for _, addr := range addresses {
				inbound.To = append(inbound.To, addr.Address)
			}
		} else {
			// Fallback: just use raw header
			inbound.To = []string{toHeader}
		}
	}

	// Read body
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read email body: %w", err)
	}

	// TODO: Handle multipart MIME to extract HTML and attachments
	// For now, just use the raw body as text
	inbound.Body = string(body)

	return inbound, nil
}

// lookupOriginalEmail finds the original email by Message-ID.
func (s *InboundEmailService) lookupOriginalEmail(ctx context.Context, messageID string) (*originalEmailInfo, error) {
	// Try PostgreSQL first (active emails)
	email, err := s.pgRepo.GetEmailByMessageID(ctx, messageID)
	if err == nil {
		return &originalEmailInfo{ID: email.ID, UserID: email.UserID}, nil
	}

	// Try ClickHouse (archived emails)
	archivedEmail, err := s.chRepo.GetEmailByMessageID(ctx, messageID)
	if err == nil {
		return &originalEmailInfo{ID: archivedEmail.ID, UserID: archivedEmail.UserID}, nil
	}

	return nil, fmt.Errorf("original email not found for message ID: %s", messageID)
}

type originalEmailInfo struct {
	ID     string
	UserID string
}

// deliverWebhook sends the reply notification to the user via Svix.
func (s *InboundEmailService) deliverWebhook(ctx context.Context, email *InboundEmail) error {
	// Ensure user has a Svix app
	if err := s.svixClient.EnsureApp(ctx, email.UserID, "User "+email.UserID); err != nil {
		return fmt.Errorf("failed to ensure svix app: %w", err)
	}

	// Build webhook payload
	payload := map[string]interface{}{
		"type": "email.reply_received",
		"data": map[string]interface{}{
			"id":                email.ID,
			"message_id":        email.MessageID,
			"in_reply_to":       email.InReplyTo,
			"references":        email.References,
			"from":              email.From,
			"to":                email.To,
			"subject":           email.Subject,
			"body":              email.Body,
			"html":              email.HTML,
			"original_email_id": email.OriginalEmailID,
		},
	}

	// Send via Svix
	err := s.svixClient.SendMessage(ctx, email.UserID, "email.reply_received", payload)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}

	return nil
}

// cleanMessageID removes angle brackets from Message-ID.
func cleanMessageID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "<")
	id = strings.TrimSuffix(id, ">")
	return id
}
