package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	v1 "github.com/emailapi/api/gen/v1"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/webhook"
)

// SNSNotificationService handles all SNS notifications from SES.
// This includes: Delivery, Bounce, Complaint, Send, Reject, DeliveryDelay.
// Uses routing table to map message_id -> user_id for webhook delivery.
type SNSNotificationService struct {
	chRepo        *chrepo.EmailRepository
	activityRepo  *chrepo.ActivityRepository
	webhookSender webhook.Sender
	suppressRepo  *suppression.Repository
}

// NewSNSNotificationService creates a new SNSNotificationService.
func NewSNSNotificationService(
	chRepo *chrepo.EmailRepository,
	activityRepo *chrepo.ActivityRepository,
	webhookSender webhook.Sender,
	suppressRepo *suppression.Repository,
) *SNSNotificationService {
	return &SNSNotificationService{
		chRepo:        chRepo,
		activityRepo:  activityRepo,
		webhookSender: webhookSender,
		suppressRepo:  suppressRepo,
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
	Token            string // For SubscriptionConfirmation/UnsubscribeConfirmation
}

// SESEventNotification represents the parsed SES notification inside SNS Message.
type SESEventNotification struct {
	NotificationType string             `json:"eventType"` // AWS uses "eventType" not "notificationType"
	Delivery         *SESEventDelivery  `json:"delivery,omitempty"`
	Bounce           *SESEventBounce    `json:"bounce,omitempty"`
	Complaint        *SESEventComplaint `json:"complaint,omitempty"`
	Send             *SESEventSend      `json:"send,omitempty"`
	Reject           *SESEventReject    `json:"reject,omitempty"`
	Mail             SESEventMail       `json:"mail"`
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

// HandleNotification processes an SNS notification from SES.
// Note: Signature verification is handled by the SNS middleware interceptor.
func (s *SNSNotificationService) HandleNotification(ctx context.Context, input *SNSInput) error {
	switch input.Type {
	case "SubscriptionConfirmation":
		return s.handleSubscriptionConfirmation(input.SubscribeURL)
	case "Notification":
		return s.handleSESEventNotification(ctx, input.Message)
	case "UnsubscribeConfirmation":
		return nil
	default:
		return fmt.Errorf("unknown SNS notification type: %s", input.Type)
	}
}

func (s *SNSNotificationService) handleSubscriptionConfirmation(subscribeURL string) error {
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
	default:
		return nil
	}
}

// handleDelivery processes email delivery confirmations.
func (s *SNSNotificationService) handleDelivery(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	// Look up routing to get user_id
	routing, err := s.chRepo.LookupRouting(ctx, messageID)
	if err != nil {
		// Can't find routing - log and skip (might be old email)
		return nil
	}

	// Log activity
	s.activityRepo.Log(ctx, routing.UserID, "email", routing.EmailID, "delivered", "success",
		fmt.Sprintf("Delivered to %v", notification.Delivery.Recipients),
		map[string]interface{}{
			"message_id": messageID,
			"recipients": notification.Delivery.Recipients,
			"timestamp":  notification.Delivery.Timestamp,
		})

	// Send webhook to user
	if s.webhookSender != nil {
		event := &v1.EmailDeliveredEvent{
			EmailId:      routing.EmailID,
			UserId:       routing.UserID,
			MessageId:    messageID,
			Recipients:   notification.Delivery.Recipients,
			SmtpResponse: notification.Delivery.SMTPResponse,
		}
		s.webhookSender.SendEmailDelivered(ctx, routing.UserID, event)
	}
	return nil
}

// handleBounce processes email bounces.
func (s *SNSNotificationService) handleBounce(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	routing, err := s.chRepo.LookupRouting(ctx, messageID)
	if err != nil {
		return nil
	}

	// Add bounced recipients to suppression list
	if s.suppressRepo != nil {
		for _, recipient := range notification.Bounce.BouncedRecipients {
			reason := suppression.ReasonBounceSoft
			if notification.Bounce.BounceType == "Permanent" {
				reason = suppression.ReasonBounceHard
			}

			if err := s.suppressRepo.Add(ctx, &suppression.Entry{
				EmailHash:       suppression.HashEmail(recipient.EmailAddress),
				UserID:          routing.UserID,
				Reason:          reason,
				BounceType:      notification.Bounce.BounceType,
				SourceMessageID: messageID,
			}); err != nil {
				fmt.Printf("Warning: failed to add %s to suppression list: %v\n", recipient.EmailAddress, err)
			}
		}
	}

	// Log activity
	bouncedRecipients := make([]string, len(notification.Bounce.BouncedRecipients))
	for i, r := range notification.Bounce.BouncedRecipients {
		bouncedRecipients[i] = r.EmailAddress
	}

	s.activityRepo.Log(ctx, routing.UserID, "email", routing.EmailID, "bounced", "failed",
		fmt.Sprintf("Bounce: %s - %s", notification.Bounce.BounceType, notification.Bounce.BounceSubType),
		map[string]interface{}{
			"message_id":     messageID,
			"bounce_type":    notification.Bounce.BounceType,
			"bounce_subtype": notification.Bounce.BounceSubType,
			"recipients":     bouncedRecipients,
		})

	// Send webhook
	if s.webhookSender != nil {
		// Collect diagnostic codes
		diagnosticCodes := make([]string, len(notification.Bounce.BouncedRecipients))
		for i, r := range notification.Bounce.BouncedRecipients {
			diagnosticCodes[i] = r.DiagnosticCode
		}

		event := &v1.EmailBouncedEvent{
			EmailId:         routing.EmailID,
			UserId:          routing.UserID,
			MessageId:       messageID,
			BounceType:      notification.Bounce.BounceType,
			BounceSubtype:   notification.Bounce.BounceSubType,
			Recipients:      bouncedRecipients,
			DiagnosticCodes: diagnosticCodes,
		}
		s.webhookSender.SendEmailBounced(ctx, routing.UserID, event)
	}
	return nil
}

// handleComplaint processes spam complaints.
func (s *SNSNotificationService) handleComplaint(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	routing, err := s.chRepo.LookupRouting(ctx, messageID)
	if err != nil {
		return nil
	}

	// Add complained recipients to suppression list (permanent)
	if s.suppressRepo != nil {
		for _, recipient := range notification.Complaint.ComplainedRecipients {
			if err := s.suppressRepo.Add(ctx, &suppression.Entry{
				EmailHash:       suppression.HashEmail(recipient.EmailAddress),
				UserID:          routing.UserID,
				Reason:          suppression.ReasonComplaint,
				SourceMessageID: messageID,
			}); err != nil {
				fmt.Printf("Warning: failed to add %s to suppression list: %v\n", recipient.EmailAddress, err)
			}
		}
	}

	complainedRecipients := make([]string, len(notification.Complaint.ComplainedRecipients))
	for i, r := range notification.Complaint.ComplainedRecipients {
		complainedRecipients[i] = r.EmailAddress
	}

	// Log activity
	s.activityRepo.Log(ctx, routing.UserID, "email", routing.EmailID, "complained", "failed",
		fmt.Sprintf("Complaint: %s", notification.Complaint.ComplaintFeedbackType),
		map[string]interface{}{
			"message_id":    messageID,
			"feedback_type": notification.Complaint.ComplaintFeedbackType,
			"recipients":    complainedRecipients,
		})

	// Send webhook
	if s.webhookSender != nil {
		event := &v1.EmailComplainedEvent{
			EmailId:      routing.EmailID,
			UserId:       routing.UserID,
			MessageId:    messageID,
			FeedbackType: notification.Complaint.ComplaintFeedbackType,
			Recipients:   complainedRecipients,
		}
		s.webhookSender.SendEmailComplained(ctx, routing.UserID, event)
	}
	return nil
}

// handleSend processes send confirmations.
func (s *SNSNotificationService) handleSend(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	routing, err := s.chRepo.LookupRouting(ctx, messageID)
	if err != nil {
		return nil
	}

	// Log activity only
	s.activityRepo.Log(ctx, routing.UserID, "email", routing.EmailID, "accepted", "success",
		"Email accepted by SES",
		map[string]interface{}{"message_id": messageID})

	return nil
}

// handleReject processes email rejections.
func (s *SNSNotificationService) handleReject(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	routing, err := s.chRepo.LookupRouting(ctx, messageID)
	if err != nil {
		return nil
	}

	// Log activity
	s.activityRepo.Log(ctx, routing.UserID, "email", routing.EmailID, "rejected", "failed",
		fmt.Sprintf("Rejected: %s", notification.Reject.Reason),
		map[string]interface{}{"message_id": messageID, "reason": notification.Reject.Reason})

	// Send webhook
	if s.webhookSender != nil {
		event := &v1.EmailRejectedEvent{
			EmailId:   routing.EmailID,
			UserId:    routing.UserID,
			MessageId: messageID,
			Reason:    notification.Reject.Reason,
		}
		s.webhookSender.SendEmailRejected(ctx, routing.UserID, event)
	}
	return nil
}

// handleDeliveryDelay logs delivery delays.
func (s *SNSNotificationService) handleDeliveryDelay(ctx context.Context, notification *SESEventNotification) error {
	messageID := notification.Mail.MessageID

	routing, err := s.chRepo.LookupRouting(ctx, messageID)
	if err != nil {
		return nil
	}

	// Log activity
	s.activityRepo.Log(ctx, routing.UserID, "email", routing.EmailID, "delayed", "warning",
		"Email delivery delayed",
		map[string]interface{}{"message_id": messageID})

	// Send webhook
	if s.webhookSender != nil {
		event := &v1.EmailDelayedEvent{
			EmailId:   routing.EmailID,
			UserId:    routing.UserID,
			MessageId: messageID,
		}
		s.webhookSender.SendEmailDelayed(ctx, routing.UserID, event)
	}
	return nil
}
