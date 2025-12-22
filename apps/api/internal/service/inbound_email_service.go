package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jordan-wright/email"

	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/sns"
	"github.com/emailapi/api/internal/external/svix"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
)

// InboundEmailService handles inbound email processing (replies) from SNS/SES.
type InboundEmailService struct {
	s3Factory   *s3.Factory
	chRepo      *chrepo.EmailRepository
	svixClient  svix.Client
	snsVerifier *sns.Verifier
}

// NewInboundEmailService creates a new InboundEmailService.
func NewInboundEmailService(
	s3Factory *s3.Factory,
	chRepo *chrepo.EmailRepository,
	svixClient svix.Client,
) *InboundEmailService {
	return &InboundEmailService{
		s3Factory:   s3Factory,
		chRepo:      chRepo,
		svixClient:  svixClient,
		snsVerifier: sns.NewVerifier(),
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
	OriginalFrom    string   `json:"original_from"`
	UserID          string   `json:"user_id"`
}

// SNSNotificationInput contains all fields from an SNS notification for verification and processing.
type SNSNotificationInput struct {
	Type             string
	MessageID        string
	TopicArn         string
	Message          string
	SubscribeURL     string
	Timestamp        string
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	Subject          string
}

// HandleSNSNotification processes SNS notifications from SES.
func (s *InboundEmailService) HandleSNSNotification(ctx context.Context, input *SNSNotificationInput) error {
	// Verify SNS signature first
	if err := s.snsVerifier.VerifySignature(
		input.SigningCertURL,
		input.Signature,
		input.SignatureVersion,
		input.Type,
		input.Message,
		input.MessageID,
		input.Timestamp,
		input.TopicArn,
		input.SubscribeURL,
		input.Subject,
	); err != nil {
		return fmt.Errorf("SNS signature verification failed: %w", err)
	}

	switch input.Type {
	case "SubscriptionConfirmation":
		return s.handleSubscriptionConfirmation(ctx, input.SubscribeURL)
	case "Notification":
		return s.handleNotification(ctx, input.Message)
	case "UnsubscribeConfirmation":
		return nil
	default:
		return fmt.Errorf("unknown SNS notification type: %s", input.Type)
	}
}

// handleSubscriptionConfirmation auto-confirms the SNS subscription.
func (s *InboundEmailService) handleSubscriptionConfirmation(ctx context.Context, subscribeURL string) error {
	if subscribeURL == "" {
		return fmt.Errorf("missing subscribe URL")
	}

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
	var sesNotif SESNotification
	if err := json.Unmarshal([]byte(message), &sesNotif); err != nil {
		fmt.Printf("[Inbound] Failed to parse SES notification: %v\n", err)
		return fmt.Errorf("failed to parse SES notification: %w", err)
	}

	if sesNotif.NotificationType != "Received" {
		return nil
	}

	if sesNotif.Receipt.Action.Type != "S3" {
		return fmt.Errorf("expected S3 action, got: %s", sesNotif.Receipt.Action.Type)
	}

	key := sesNotif.Receipt.Action.ObjectKey
	rawEmail, err := s.s3Factory.Bucket(s3.BucketInbound).Download(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to download email from S3: %w", err)
	}

	inboundEmail, err := s.parseEmail(rawEmail)
	if err != nil {
		return fmt.Errorf("failed to parse email: %w", err)
	}

	// Look up original email by In-Reply-To header
	if inboundEmail.InReplyTo == "" {
		if len(inboundEmail.References) == 0 {
			return nil // Not a reply
		}
		inboundEmail.InReplyTo = inboundEmail.References[len(inboundEmail.References)-1]
	}

	// Use routing table as single source of truth
	routing, err := s.chRepo.LookupRouting(ctx, inboundEmail.InReplyTo)
	if err != nil {
		return nil // Reply to email we didn't send (not in our routing table)
	}

	inboundEmail.OriginalEmailID = routing.EmailID
	inboundEmail.OriginalFrom = routing.FromEmail
	inboundEmail.UserID = routing.UserID

	// Log the reply event to ClickHouse
	if err := s.chRepo.AddReplyEvent(
		ctx,
		routing.EmailID,
		inboundEmail.MessageID,
		routing.UserID,
		inboundEmail.From,
		inboundEmail.Subject,
		map[string]string{
			"in_reply_to": inboundEmail.InReplyTo,
		},
	); err != nil {
		fmt.Printf("Warning: failed to log reply event: %v\n", err)
	}

	return s.deliverWebhook(ctx, inboundEmail)
}

// parseEmail parses raw MIME email using jordan-wright/email package.
func (s *InboundEmailService) parseEmail(rawEmail []byte) (*InboundEmail, error) {
	parsed, err := email.NewEmailFromReader(bytes.NewReader(rawEmail))
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}

	inbound := &InboundEmail{
		ID:        uuid.New().String(),
		MessageID: cleanMessageID(parsed.Headers.Get("Message-ID")),
		InReplyTo: cleanMessageID(parsed.Headers.Get("In-Reply-To")),
		From:      parsed.From,
		Subject:   parsed.Subject,
		Body:      string(parsed.Text),
		HTML:      string(parsed.HTML),
	}

	// Parse References header
	if references := parsed.Headers.Get("References"); references != "" {
		for _, ref := range strings.Fields(references) {
			inbound.References = append(inbound.References, cleanMessageID(ref))
		}
	}

	// Parse To addresses
	for _, to := range parsed.To {
		if addr, err := mail.ParseAddress(to); err == nil {
			inbound.To = append(inbound.To, addr.Address)
		} else {
			inbound.To = append(inbound.To, to)
		}
	}

	return inbound, nil
}

// deliverWebhook sends the reply notification to the user via Svix.
func (s *InboundEmailService) deliverWebhook(ctx context.Context, email *InboundEmail) error {
	if err := s.svixClient.EnsureApp(ctx, email.UserID, "User "+email.UserID); err != nil {
		return fmt.Errorf("failed to ensure svix app: %w", err)
	}

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
			"original_from":     email.OriginalFrom, // Original sender for reply-to translation
		},
	}

	if err := s.svixClient.SendMessage(ctx, email.UserID, "email.reply_received", payload); err != nil {
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
