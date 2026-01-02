package service

import (
	"bytes"
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jordan-wright/email"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/autoresponse"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/webhook"
)

// InboundEmailService handles inbound email processing (replies).
// It now uses SQS event processing via ProcessRawEmail instead of SNS webhooks.
type InboundEmailService struct {
	s3Factory     s3.FactoryInterface
	analytics     Analytics
	webhookSender webhook.Sender
}

// NewInboundEmailService creates a new InboundEmailService.
func NewInboundEmailService(
	s3Factory s3.FactoryInterface,
	analytics Analytics,
	webhookSender webhook.Sender,
) *InboundEmailService {
	return &InboundEmailService{
		s3Factory:     s3Factory,
		analytics:     analytics,
		webhookSender: webhookSender,
	}
}

// InboundEmail represents a parsed inbound email.
type InboundEmail struct {
	ID               string   `json:"id"`
	MessageID        string   `json:"message_id"`
	InReplyTo        string   `json:"in_reply_to"`
	References       []string `json:"references"`
	From             string   `json:"from"`
	To               []string `json:"to"`
	Subject          string   `json:"subject"`
	Body             string   `json:"body"`
	HTML             string   `json:"html"`
	OriginalEmailID  string   `json:"original_email_id"`
	UserID           string   `json:"user_id"`
	IsAutoResponse   bool     `json:"is_auto_response"`
	AutoResponseType string   `json:"auto_response_type,omitempty"`
	AutoResponseInfo string   `json:"auto_response_info,omitempty"`
}

// ProcessRawEmail processes a raw email from S3 (for SQS-based processing).
// This is the main entry point for the new S3 → EventBridge → SQS flow.
func (s *InboundEmailService) ProcessRawEmail(ctx context.Context, bucket, key string) error {
	// Download email from S3
	rawEmail, err := s.s3Factory.Bucket(s3.BucketInbound).Download(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to download email from S3: %w", err)
	}

	// Parse email and check for virus/spam
	inboundEmail, virusVerdict, spamVerdict, err := s.parseEmailWithVerdicts(rawEmail)
	if err != nil {
		return fmt.Errorf("failed to parse email: %w", err)
	}

	// Check virus verdict (SES adds X-SES-Virus-Verdict header)
	if virusVerdict == "FAIL" {
		fmt.Printf("Inbound email %s rejected: virus detected\n", key)
		// Log activity if we can find routing
		s.logSecurityEvent(ctx, inboundEmail, "virus_rejected", "Email rejected due to virus detection")
		return nil // Don't process, but don't error
	}

	// Log spam verdict (process anyway, but log it)
	if spamVerdict == "FAIL" {
		fmt.Printf("Inbound email %s flagged as spam, processing anyway\n", key)
		s.logSecurityEvent(ctx, inboundEmail, "spam_flagged", "Email flagged as spam but processed")
	}

	// Process the email (same as handleNotification)
	return s.processInboundEmail(ctx, inboundEmail)
}

// logSecurityEvent logs virus/spam events for inbound emails.
func (s *InboundEmailService) logSecurityEvent(ctx context.Context, email *InboundEmail, action, message string) {
	if s.analytics == nil {
		return
	}

	// Try to find the original email this is replying to
	replyTo := email.InReplyTo
	if replyTo == "" && len(email.References) > 0 {
		replyTo = email.References[len(email.References)-1]
	}

	if replyTo == "" {
		// Not a reply - log as system event
		s.analytics.Activity().Log(ctx, "", "inbound_email", email.ID, action, "blocked",
			message,
			map[string]interface{}{
				"from":       email.From,
				"subject":    email.Subject,
				"message_id": email.MessageID,
			})
		return
	}

	routing, err := s.analytics.Email().LookupRouting(ctx, replyTo)
	if err != nil {
		// Can't find original email - still log it
		s.analytics.Activity().Log(ctx, "", "inbound_email", email.ID, action, "blocked",
			message,
			map[string]interface{}{
				"from":        email.From,
				"subject":     email.Subject,
				"message_id":  email.MessageID,
				"in_reply_to": replyTo,
			})
		return
	}

	// Log with user context
	s.analytics.Activity().Log(ctx, routing.UserID, "email", routing.EmailID, action, "blocked",
		fmt.Sprintf("%s from %s: %s", message, email.From, email.Subject),
		map[string]interface{}{
			"inbound_email_id":  email.ID,
			"from":              email.From,
			"subject":           email.Subject,
			"message_id":        email.MessageID,
			"original_email_id": routing.EmailID,
		})
}

// processInboundEmail handles the core logic for inbound emails.
func (s *InboundEmailService) processInboundEmail(ctx context.Context, inboundEmail *InboundEmail) error {
	// Look up original email by In-Reply-To header
	if inboundEmail.InReplyTo == "" {
		if len(inboundEmail.References) == 0 {
			return nil // Not a reply
		}
		inboundEmail.InReplyTo = inboundEmail.References[len(inboundEmail.References)-1]
	}

	// Use routing table as single source of truth
	if s.analytics == nil {
		return nil
	}
	routing, err := s.analytics.Email().LookupRouting(ctx, inboundEmail.InReplyTo)
	if err != nil {
		return nil // Reply to email we didn't send
	}

	inboundEmail.OriginalEmailID = routing.EmailID
	inboundEmail.UserID = routing.UserID

	// Add inbound email to routing table
	if err := s.analytics.Email().InsertRouting(ctx, inboundEmail.MessageID, inboundEmail.ID, inboundEmail.UserID); err != nil {
		fmt.Printf("Warning: failed to insert routing entry for inbound email: %v\n", err)
	}

	// Check if this is an auto-response
	if inboundEmail.IsAutoResponse {
		s.analytics.Email().LogEmailEvent(
			ctx,
			routing.UserID,
			routing.EmailID,
			"auto_response",
			"ignored",
			fmt.Sprintf("Auto-response (%s) from %s: %s", inboundEmail.AutoResponseType, inboundEmail.From, inboundEmail.Subject),
			map[string]interface{}{
				"inbound_email_id":     inboundEmail.ID,
				"auto_response_type":   inboundEmail.AutoResponseType,
				"auto_response_reason": inboundEmail.AutoResponseInfo,
			},
		)
		return nil
	}

	// Log activity
	s.analytics.Activity().Log(
		ctx,
		routing.UserID,
		"email",
		routing.EmailID,
		"replied",
		"success",
		fmt.Sprintf("Reply received from %s: %s", inboundEmail.From, inboundEmail.Subject),
		map[string]interface{}{
			"inbound_email_id": inboundEmail.ID,
			"message_id":       inboundEmail.MessageID,
			"from":             inboundEmail.From,
			"subject":          inboundEmail.Subject,
		},
	)

	return s.deliverWebhook(ctx, inboundEmail)
}

// parseEmailWithVerdicts parses email and extracts SES verdict headers.
func (s *InboundEmailService) parseEmailWithVerdicts(rawEmail []byte) (*InboundEmail, string, string, error) {
	parsed, err := email.NewEmailFromReader(bytes.NewReader(rawEmail))
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to parse email: %w", err)
	}

	// Extract SES verdict headers
	virusVerdict := parsed.Headers.Get("X-SES-Virus-Verdict")
	spamVerdict := parsed.Headers.Get("X-SES-Spam-Verdict")

	// Detect auto-responses (OOO, vacation, bounces, etc.)
	detector := autoresponse.NewDetector()
	autoInfo := detector.Detect(parsed.Headers)

	inbound := &InboundEmail{
		ID:               uuid.New().String(),
		MessageID:        cleanMessageID(parsed.Headers.Get("Message-ID")),
		InReplyTo:        cleanMessageID(parsed.Headers.Get("In-Reply-To")),
		From:             parsed.From,
		Subject:          parsed.Subject,
		Body:             string(parsed.Text),
		HTML:             string(parsed.HTML),
		IsAutoResponse:   autoInfo.IsAutoResponse,
		AutoResponseType: string(autoInfo.Type),
		AutoResponseInfo: autoInfo.Reason,
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

	return inbound, virusVerdict, spamVerdict, nil
}

// parseEmail parses raw MIME email (legacy, kept for backward compatibility).
func (s *InboundEmailService) parseEmail(rawEmail []byte) (*InboundEmail, error) {
	inbound, _, _, err := s.parseEmailWithVerdicts(rawEmail)
	return inbound, err
}

// deliverWebhook sends the reply notification to the user via webhook.Sender.
func (s *InboundEmailService) deliverWebhook(ctx context.Context, email *InboundEmail) error {
	if s.webhookSender == nil {
		return nil
	}

	event := &v1.EmailRepliedEvent{
		Id:            email.ID,
		UserId:        email.UserID,
		MessageId:     email.MessageID,
		InReplyTo:     email.InReplyTo,
		References:    email.References,
		From:          email.From,
		To:            email.To,
		Subject:       email.Subject,
		Body:          email.Body,
		Html:          email.HTML,
		ParentEmailId: email.OriginalEmailID,
	}

	s.webhookSender.SendEmailReplied(ctx, email.UserID, event)
	return nil
}

// cleanMessageID removes angle brackets from Message-ID.
func cleanMessageID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "<")
	id = strings.TrimSuffix(id, ">")
	return id
}
