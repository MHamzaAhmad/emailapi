package handlers

import (
	"context"
	"fmt"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/events"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/repository/tinybird"
	"github.com/emailapi/api/internal/webhook"
)

// Dependencies contains shared dependencies for all handlers.
type Dependencies struct {
	Analytics     Analytics
	WebhookSender webhook.Sender
	SuppressRepo  SuppressionManager
	ReputationSvc ReputationRecorder
}

// Analytics interface for activity logging and email routing.
type Analytics interface {
	Activity() ActivityLogger
	Email() EmailRouter
}

// ActivityLogger logs user activities.
type ActivityLogger interface {
	Log(ctx context.Context, userID, entityType, entityID, action, status, message string, metadata map[string]interface{}) error
}

// EmailRouter looks up email routing information.
type EmailRouter interface {
	LookupRouting(ctx context.Context, messageID string) (*tinybird.EmailRouting, error)
}

// SuppressionManager manages email suppression lists.
type SuppressionManager interface {
	Add(ctx context.Context, entry *suppression.Entry) error
}

// ReputationRecorder records reputation incidents.
type ReputationRecorder interface {
	RecordBounceIncident(ctx context.Context, userID, messageID, bounceType, bounceSubType string, recipients []domain.BounceRecipient) error
	RecordComplaintIncident(ctx context.Context, userID, messageID, feedbackType string, recipients []string) error
}

// BounceHandler processes bounce events.
type BounceHandler struct {
	deps *Dependencies
}

// NewBounceHandler creates a new bounce handler.
func NewBounceHandler(deps *Dependencies) *BounceHandler {
	return &BounceHandler{deps: deps}
}

// Handle processes a bounce event.
func (h *BounceHandler) Handle(ctx context.Context, event *events.SESEvent) error {
	if event.Bounce == nil {
		return fmt.Errorf("bounce event missing bounce data")
	}

	messageID := event.Mail.MessageId
	isHardBounce := event.Bounce.BounceType == "Permanent"

	// Add hard bounces to global suppression
	if h.deps.SuppressRepo != nil && isHardBounce {
		for _, recipient := range event.Bounce.BouncedRecipients {
			_ = h.deps.SuppressRepo.Add(ctx, &suppression.Entry{
				EmailHash:       suppression.HashEmail(recipient.EmailAddress),
				UserID:          "", // Global
				Reason:          suppression.ReasonBounceHard,
				BounceType:      event.Bounce.BounceType,
				SourceMessageID: messageID,
			})
		}
	}

	// Get routing for activity/webhook
	routing, err := h.deps.Analytics.Email().LookupRouting(ctx, messageID)
	if err != nil {
		return nil // No routing - suppression added, skip rest
	}

	// Add soft bounces to per-user suppression
	if h.deps.SuppressRepo != nil && !isHardBounce {
		for _, recipient := range event.Bounce.BouncedRecipients {
			_ = h.deps.SuppressRepo.Add(ctx, &suppression.Entry{
				EmailHash:       suppression.HashEmail(recipient.EmailAddress),
				UserID:          routing.UserID,
				Reason:          suppression.ReasonBounceSoft,
				BounceType:      event.Bounce.BounceType,
				SourceMessageID: messageID,
			})
		}
	}

	// Log activity
	recipients := make([]string, len(event.Bounce.BouncedRecipients))
	for i, r := range event.Bounce.BouncedRecipients {
		recipients[i] = r.EmailAddress
	}

	h.deps.Analytics.Activity().Log(ctx, routing.UserID, "email", routing.EmailID, "bounced", "failed",
		fmt.Sprintf("Bounce: %s - %s", event.Bounce.BounceType, event.Bounce.BounceSubType),
		map[string]interface{}{
			"message_id":     messageID,
			"bounce_type":    event.Bounce.BounceType,
			"bounce_subtype": event.Bounce.BounceSubType,
			"recipients":     recipients,
		})

	// Send webhook
	if h.deps.WebhookSender != nil {
		diagnosticCodes := make([]string, len(event.Bounce.BouncedRecipients))
		for i, r := range event.Bounce.BouncedRecipients {
			diagnosticCodes[i] = r.DiagnosticCode
		}

		webhookEvent := &v1.EmailBouncedEvent{
			EmailId:         routing.EmailID,
			UserId:          routing.UserID,
			MessageId:       messageID,
			BounceType:      event.Bounce.BounceType,
			BounceSubtype:   event.Bounce.BounceSubType,
			Recipients:      recipients,
			DiagnosticCodes: diagnosticCodes,
		}
		h.deps.WebhookSender.SendEmailBounced(ctx, routing.UserID, webhookEvent)
	}

	// Record reputation incident
	if h.deps.ReputationSvc != nil {
		domainRecipients := make([]domain.BounceRecipient, len(event.Bounce.BouncedRecipients))
		for i, r := range event.Bounce.BouncedRecipients {
			domainRecipients[i] = domain.BounceRecipient{
				EmailAddress:   r.EmailAddress,
				DiagnosticCode: r.DiagnosticCode,
			}
		}
		go func() {
			_ = h.deps.ReputationSvc.RecordBounceIncident(
				context.Background(),
				routing.UserID,
				messageID,
				event.Bounce.BounceType,
				event.Bounce.BounceSubType,
				domainRecipients,
			)
		}()
	}

	return nil
}

// ComplaintHandler processes complaint events.
type ComplaintHandler struct {
	deps *Dependencies
}

// NewComplaintHandler creates a new complaint handler.
func NewComplaintHandler(deps *Dependencies) *ComplaintHandler {
	return &ComplaintHandler{deps: deps}
}

// Handle processes a complaint event.
func (h *ComplaintHandler) Handle(ctx context.Context, event *events.SESEvent) error {
	if event.Complaint == nil {
		return fmt.Errorf("complaint event missing complaint data")
	}

	messageID := event.Mail.MessageId

	// Add to global suppression
	if h.deps.SuppressRepo != nil {
		for _, recipient := range event.Complaint.ComplainedRecipients {
			_ = h.deps.SuppressRepo.Add(ctx, &suppression.Entry{
				EmailHash:       suppression.HashEmail(recipient.EmailAddress),
				UserID:          "", // Global
				Reason:          suppression.ReasonComplaint,
				SourceMessageID: messageID,
			})
		}
	}

	routing, err := h.deps.Analytics.Email().LookupRouting(ctx, messageID)
	if err != nil {
		return nil
	}

	recipients := make([]string, len(event.Complaint.ComplainedRecipients))
	for i, r := range event.Complaint.ComplainedRecipients {
		recipients[i] = r.EmailAddress
	}

	// Log activity
	h.deps.Analytics.Activity().Log(ctx, routing.UserID, "email", routing.EmailID, "complained", "failed",
		fmt.Sprintf("Complaint: %s", event.Complaint.ComplaintFeedbackType),
		map[string]interface{}{
			"message_id":    messageID,
			"feedback_type": event.Complaint.ComplaintFeedbackType,
			"recipients":    recipients,
		})

	// Webhook
	if h.deps.WebhookSender != nil {
		webhookEvent := &v1.EmailComplainedEvent{
			EmailId:      routing.EmailID,
			UserId:       routing.UserID,
			MessageId:    messageID,
			FeedbackType: event.Complaint.ComplaintFeedbackType,
			Recipients:   recipients,
		}
		h.deps.WebhookSender.SendEmailComplained(ctx, routing.UserID, webhookEvent)
	}

	// Reputation
	if h.deps.ReputationSvc != nil {
		go func() {
			_ = h.deps.ReputationSvc.RecordComplaintIncident(
				context.Background(),
				routing.UserID,
				messageID,
				event.Complaint.ComplaintFeedbackType,
				recipients,
			)
		}()
	}

	return nil
}

// DeliveryHandler processes delivery confirmation events.
type DeliveryHandler struct {
	deps *Dependencies
}

// NewDeliveryHandler creates a new delivery handler.
func NewDeliveryHandler(deps *Dependencies) *DeliveryHandler {
	return &DeliveryHandler{deps: deps}
}

// Handle processes a delivery event.
func (h *DeliveryHandler) Handle(ctx context.Context, event *events.SESEvent) error {
	if event.Delivery == nil {
		return fmt.Errorf("delivery event missing delivery data")
	}

	messageID := event.Mail.MessageId

	routing, err := h.deps.Analytics.Email().LookupRouting(ctx, messageID)
	if err != nil {
		return nil
	}

	// Log activity
	h.deps.Analytics.Activity().Log(ctx, routing.UserID, "email", routing.EmailID, "delivered", "success",
		fmt.Sprintf("Delivered to %v", event.Delivery.Recipients),
		map[string]interface{}{
			"message_id": messageID,
			"recipients": event.Delivery.Recipients,
			"timestamp":  event.Delivery.Timestamp,
		})

	// Webhook
	if h.deps.WebhookSender != nil {
		webhookEvent := &v1.EmailDeliveredEvent{
			EmailId:      routing.EmailID,
			UserId:       routing.UserID,
			MessageId:    messageID,
			Recipients:   event.Delivery.Recipients,
			SmtpResponse: event.Delivery.SmtpResponse,
		}
		h.deps.WebhookSender.SendEmailDelivered(ctx, routing.UserID, webhookEvent)
	}

	return nil
}
