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

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/autoresponse"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/webhook"
)

// InboundEmailService handles inbound email processing (replies) from SNS/SES.
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
	Token            string // For SubscriptionConfirmation/UnsubscribeConfirmation
}

// HandleSNSNotification processes SNS notifications from SES for inbound emails.
// Note: Signature verification is handled by the SNS middleware interceptor.
func (s *InboundEmailService) HandleSNSNotification(ctx context.Context, input *SNSNotificationInput) error {
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
	if s.analytics == nil {
		return nil // Cannot lookup without analytics
	}
	routing, err := s.analytics.Email().LookupRouting(ctx, inboundEmail.InReplyTo)
	if err != nil {
		return nil // Reply to email we didn't send (not in our routing table)
	}

	inboundEmail.OriginalEmailID = routing.EmailID
	inboundEmail.UserID = routing.UserID

	// Add inbound email to routing table so replies to this email can be tracked too
	if err := s.analytics.Email().InsertRouting(ctx, inboundEmail.MessageID, inboundEmail.ID, inboundEmail.UserID); err != nil {
		fmt.Printf("Warning: failed to insert routing entry for inbound email: %v\n", err)
	}

	// Check if this is an auto-response (OOO, vacation, bounce, etc.)
	if inboundEmail.IsAutoResponse {
		// Log auto-response but skip webhook delivery
		s.analytics.Email().LogEmailEvent(
			ctx,
			routing.UserID,
			routing.EmailID,
			"auto_response",
			"ignored",
			fmt.Sprintf("Auto-response (%s) from %s: %s", inboundEmail.AutoResponseType, inboundEmail.From, inboundEmail.Subject),
			map[string]interface{}{
				"inbound_email_id":     inboundEmail.ID,
				"message_id":           inboundEmail.MessageID,
				"from":                 inboundEmail.From,
				"subject":              inboundEmail.Subject,
				"auto_response_type":   inboundEmail.AutoResponseType,
				"auto_response_reason": inboundEmail.AutoResponseInfo,
			},
		)
		return nil // Skip webhook for auto-responses
	}

	// Log the reply event to ClickHouse activity_logs
	s.analytics.Email().LogEmailEvent(
		ctx,
		routing.UserID,
		routing.EmailID,
		"replied",
		"success",
		fmt.Sprintf("Reply from %s: %s", inboundEmail.From, inboundEmail.Subject),
		map[string]interface{}{
			"inbound_email_id": inboundEmail.ID,
			"message_id":       inboundEmail.MessageID,
			"from":             inboundEmail.From,
			"subject":          inboundEmail.Subject,
		},
	)

	return s.deliverWebhook(ctx, inboundEmail)
}

// parseEmail parses raw MIME email using jordan-wright/email package.
func (s *InboundEmailService) parseEmail(rawEmail []byte) (*InboundEmail, error) {
	parsed, err := email.NewEmailFromReader(bytes.NewReader(rawEmail))
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %w", err)
	}

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

	return inbound, nil
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
