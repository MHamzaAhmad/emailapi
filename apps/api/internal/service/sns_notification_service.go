package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/sns"
	"github.com/emailapi/api/internal/external/svix"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	pgrepo "github.com/emailapi/api/internal/repository/postgres"
	"github.com/emailapi/api/internal/repository/suppression"
)

// SNSNotificationService handles all SNS notifications from SES.
// This includes: Delivery, Bounce, Complaint, Send, Reject, DeliveryDelay, and inbound emails.
type SNSNotificationService struct {
	pgRepo       *pgrepo.EmailRepository
	chRepo       *chrepo.EmailRepository
	activityRepo *chrepo.ActivityRepository
	svixClient   svix.Client
	s3Factory    *s3.Factory
	suppressRepo *suppression.Repository
	snsVerifier  *sns.Verifier
}

// NewSNSNotificationService creates a new SNSNotificationService.
func NewSNSNotificationService(
	pgRepo *pgrepo.EmailRepository,
	chRepo *chrepo.EmailRepository,
	activityRepo *chrepo.ActivityRepository,
	svixClient svix.Client,
	s3Factory *s3.Factory,
	suppressRepo *suppression.Repository,
) *SNSNotificationService {
	return &SNSNotificationService{
		pgRepo:       pgRepo,
		chRepo:       chRepo,
		activityRepo: activityRepo,
		svixClient:   svixClient,
		s3Factory:    s3Factory,
		suppressRepo: suppressRepo,
		snsVerifier:  sns.NewVerifier(),
	}
}

// SNSInput contains all fields from an SNS HTTP payload.
type SNSInput struct {
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

// SESEventNotification represents the parsed SES notification inside SNS Message.
// Named differently from InboundEmailService.SESEventNotification to avoid conflicts.
type SESEventNotification struct {
	NotificationType string `json:"notificationType"`

	// For Delivery notifications
	Delivery *SESEventDelivery `json:"delivery,omitempty"`

	// For Bounce notifications
	Bounce *SESEventBounce `json:"bounce,omitempty"`

	// For Complaint notifications
	Complaint *SESEventComplaint `json:"complaint,omitempty"`

	// For Send/Reject notifications
	Send   *SESEventSend   `json:"send,omitempty"`
	Reject *SESEventReject `json:"reject,omitempty"`

	// For Received (inbound) notifications
	Receipt *SESEventReceipt `json:"receipt,omitempty"`

	// Common mail object
	Mail SESEventMail `json:"mail"`
}

type SESEventMail struct {
	MessageID   string   `json:"messageId"`
	Source      string   `json:"source"`
	Destination []string `json:"destination"`
	Timestamp   string   `json:"timestamp"`
}

type SESEventDelivery struct {
	Timestamp        string   `json:"timestamp"`
	Recipients       []string `json:"recipients"`
	ProcessingTimeMs int      `json:"processingTimeMillis"`
	SMTPResponse     string   `json:"smtpResponse"`
}

type SESEventBounce struct {
	BounceType        string                    `json:"bounceType"`
	BounceSubType     string                    `json:"bounceSubType"`
	BouncedRecipients []SESEventBounceRecipient `json:"bouncedRecipients"`
	Timestamp         string                    `json:"timestamp"`
}

type SESEventBounceRecipient struct {
	EmailAddress   string `json:"emailAddress"`
	Action         string `json:"action"`
	Status         string `json:"status"`
	DiagnosticCode string `json:"diagnosticCode"`
}

type SESEventComplaint struct {
	ComplainedRecipients  []SESEventComplaintRecipient `json:"complainedRecipients"`
	ComplaintFeedbackType string                       `json:"complaintFeedbackType"`
	Timestamp             string                       `json:"timestamp"`
}

type SESEventComplaintRecipient struct {
	EmailAddress string `json:"emailAddress"`
}

type SESEventSend struct {
	Timestamp string `json:"timestamp"`
}

type SESEventReject struct {
	Reason string `json:"reason"`
}

type SESEventReceipt struct {
	Action struct {
		Type       string `json:"type"`
		BucketName string `json:"bucketName"`
		ObjectKey  string `json:"objectKey"`
	} `json:"action"`
}

// HandleNotification processes an SNS notification from SES.
func (s *SNSNotificationService) HandleNotification(ctx context.Context, input *SNSInput) error {
	// 1. Verify SNS signature
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

	// 2. Handle by notification type
	switch input.Type {
	case "SubscriptionConfirmation":
		return s.handleSubscriptionConfirmation(ctx, input.SubscribeURL)
	case "Notification":
		return s.handleSESEventNotification(ctx, input.Message)
	case "UnsubscribeConfirmation":
		return nil
	default:
		return fmt.Errorf("unknown SNS notification type: %s", input.Type)
	}
}

func (s *SNSNotificationService) handleSubscriptionConfirmation(ctx context.Context, subscribeURL string) error {
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

func (s *SNSNotificationService) handleSESEventNotification(ctx context.Context, message string) error {
	var notification SESEventNotification
	if err := json.Unmarshal([]byte(message), &notification); err != nil {
		return fmt.Errorf("failed to parse SES notification: %w", err)
	}

	switch notification.NotificationType {
	case "Delivery":
		return s.handleDelivery(ctx, &notification)
	case "Bounce":
		return s.handleBounce(ctx, &notification)
	case "Complaint":
		return s.handleComplaint(ctx, &notification)
	case "Send":
		return s.handleSend(ctx, &notification)
	case "Reject":
		return s.handleReject(ctx, &notification)
	case "DeliveryDelay":
		return s.handleDeliveryDelay(ctx, &notification)
	case "Received":
		return s.handleReceived(ctx, &notification)
	default:
		// Unknown notification type, log and ignore
		return nil
	}
}

// handleDelivery processes email delivery confirmations.
func (s *SNSNotificationService) handleDelivery(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	// Look up email by provider message ID
	email, err := s.pgRepo.GetEmailByMessageID(ctx, messageID)
	if err != nil {
		// Email not found in active emails, might already be archived
		return nil
	}

	// Update status to delivered
	email.Status = domain.EmailStatusSent // or a new "delivered" status if you add one

	// Archive to ClickHouse
	if err := s.chRepo.SaveEmail(ctx, email); err != nil {
		fmt.Printf("Warning: failed to archive email: %v\n", err)
	}

	// Log activity
	s.chRepo.LogEmailEvent(ctx, email, "delivered")

	// Delete from PostgreSQL
	if err := s.pgRepo.Delete(ctx, email.ID); err != nil {
		fmt.Printf("Warning: failed to delete email from PG: %v\n", err)
	}

	// Send webhook to user
	return s.sendWebhook(ctx, email.UserID, "email.delivered", map[string]interface{}{
		"email_id":   email.ID,
		"message_id": messageID,
		"recipients": notification.Delivery.Recipients,
		"timestamp":  notification.Delivery.Timestamp,
	})
}

// handleBounce processes email bounces.
func (s *SNSNotificationService) handleBounce(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	email, err := s.pgRepo.GetEmailByMessageID(ctx, messageID)
	if err != nil {
		return nil
	}

	email.Status = domain.EmailStatusFailed
	email.ErrorMessage = fmt.Sprintf("Bounced: %s - %s", notification.Bounce.BounceType, notification.Bounce.BounceSubType)

	// Add bounced recipients to suppression list
	if s.suppressRepo != nil {
		for _, recipient := range notification.Bounce.BouncedRecipients {
			reason := suppression.ReasonBounceSoft
			if notification.Bounce.BounceType == "Permanent" {
				reason = suppression.ReasonBounceHard
			}

			if err := s.suppressRepo.Add(ctx, &suppression.Entry{
				EmailHash:       suppression.HashEmail(recipient.EmailAddress),
				UserID:          email.UserID,
				Reason:          reason,
				BounceType:      notification.Bounce.BounceType,
				SourceMessageID: messageID,
			}); err != nil {
				fmt.Printf("Warning: failed to add %s to suppression list: %v\n", recipient.EmailAddress, err)
			}
		}
	}

	// Archive to ClickHouse
	if err := s.chRepo.SaveEmail(ctx, email); err != nil {
		fmt.Printf("Warning: failed to archive email: %v\n", err)
	}

	// Log activity
	s.chRepo.LogEmailEvent(ctx, email, "bounced")

	// Delete from PostgreSQL
	s.pgRepo.Delete(ctx, email.ID)

	// Send webhook
	bouncedRecipients := make([]string, len(notification.Bounce.BouncedRecipients))
	for i, r := range notification.Bounce.BouncedRecipients {
		bouncedRecipients[i] = r.EmailAddress
	}

	return s.sendWebhook(ctx, email.UserID, "email.bounced", map[string]interface{}{
		"email_id":       email.ID,
		"message_id":     messageID,
		"bounce_type":    notification.Bounce.BounceType,
		"bounce_subtype": notification.Bounce.BounceSubType,
		"recipients":     bouncedRecipients,
		"timestamp":      notification.Bounce.Timestamp,
	})
}

// handleComplaint processes spam complaints.
func (s *SNSNotificationService) handleComplaint(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	email, err := s.pgRepo.GetEmailByMessageID(ctx, messageID)
	if err != nil {
		return nil
	}

	email.Status = domain.EmailStatusFailed
	email.ErrorMessage = fmt.Sprintf("Complaint: %s", notification.Complaint.ComplaintFeedbackType)

	// Add complained recipients to suppression list (permanent)
	if s.suppressRepo != nil {
		for _, recipient := range notification.Complaint.ComplainedRecipients {
			if err := s.suppressRepo.Add(ctx, &suppression.Entry{
				EmailHash:       suppression.HashEmail(recipient.EmailAddress),
				UserID:          email.UserID,
				Reason:          suppression.ReasonComplaint,
				SourceMessageID: messageID,
			}); err != nil {
				fmt.Printf("Warning: failed to add %s to suppression list: %v\n", recipient.EmailAddress, err)
			}
		}
	}

	// Archive to ClickHouse
	if err := s.chRepo.SaveEmail(ctx, email); err != nil {
		fmt.Printf("Warning: failed to archive email: %v\n", err)
	}

	// Log activity
	s.chRepo.LogEmailEvent(ctx, email, "complained")

	// Delete from PostgreSQL
	s.pgRepo.Delete(ctx, email.ID)

	// Send webhook
	complainedRecipients := make([]string, len(notification.Complaint.ComplainedRecipients))
	for i, r := range notification.Complaint.ComplainedRecipients {
		complainedRecipients[i] = r.EmailAddress
	}

	return s.sendWebhook(ctx, email.UserID, "email.complained", map[string]interface{}{
		"email_id":      email.ID,
		"message_id":    messageID,
		"feedback_type": notification.Complaint.ComplaintFeedbackType,
		"recipients":    complainedRecipients,
		"timestamp":     notification.Complaint.Timestamp,
	})
}

// handleSend processes send confirmations (email accepted by SES).
func (s *SNSNotificationService) handleSend(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	email, err := s.pgRepo.GetEmailByMessageID(ctx, messageID)
	if err != nil {
		return nil
	}

	// Just log the activity, don't archive yet (wait for delivery/bounce/complaint)
	s.chRepo.LogEmailEvent(ctx, email, "accepted")

	return nil
}

// handleReject processes email rejections.
func (s *SNSNotificationService) handleReject(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	email, err := s.pgRepo.GetEmailByMessageID(ctx, messageID)
	if err != nil {
		return nil
	}

	email.Status = domain.EmailStatusFailed
	email.ErrorMessage = fmt.Sprintf("Rejected: %s", notification.Reject.Reason)

	// Archive to ClickHouse
	s.chRepo.SaveEmail(ctx, email)
	s.chRepo.LogEmailEvent(ctx, email, "rejected")
	s.pgRepo.Delete(ctx, email.ID)

	return s.sendWebhook(ctx, email.UserID, "email.rejected", map[string]interface{}{
		"email_id":   email.ID,
		"message_id": messageID,
		"reason":     notification.Reject.Reason,
	})
}

// handleDeliveryDelay logs delivery delays.
func (s *SNSNotificationService) handleDeliveryDelay(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	email, err := s.pgRepo.GetEmailByMessageID(ctx, messageID)
	if err != nil {
		return nil
	}

	// Log activity but don't change status
	s.activityRepo.LogEmail(ctx, email.UserID, email.ID, "delayed", "warning",
		"Email delivery delayed", map[string]interface{}{
			"message_id": messageID,
			"timestamp":  time.Now().Format(time.RFC3339),
		})

	return s.sendWebhook(ctx, email.UserID, "email.delayed", map[string]interface{}{
		"email_id":   email.ID,
		"message_id": messageID,
	})
}

// handleReceived processes inbound emails.
func (s *SNSNotificationService) handleReceived(ctx context.Context, notification *SESEventNotification) error {
	if notification.Receipt == nil || notification.Receipt.Action.Type != "S3" {
		return fmt.Errorf("expected S3 action for inbound email")
	}

	// This will be handled by the existing InboundEmailService
	// For now, just log that we received it
	// The actual processing logic can be moved here later

	return nil
}

func (s *SNSNotificationService) sendWebhook(ctx context.Context, userID, eventType string, data map[string]interface{}) error {
	if s.svixClient == nil {
		return nil
	}

	if err := s.svixClient.EnsureApp(ctx, userID, "User "+userID); err != nil {
		return fmt.Errorf("failed to ensure svix app: %w", err)
	}

	payload := map[string]interface{}{
		"type": eventType,
		"data": data,
	}

	if err := s.svixClient.SendMessage(ctx, userID, eventType, payload); err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}

	return nil
}
