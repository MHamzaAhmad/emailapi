package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/external/svix"
)

// Event type constants matching Svix event types.
const (
	EventEmailSent          = "email.sent"
	EventEmailDelivered     = "email.delivered"
	EventEmailBounced       = "email.bounced"
	EventEmailComplained    = "email.complained"
	EventEmailRejected      = "email.rejected"
	EventEmailDelayed       = "email.delayed"
	EventEmailFailed        = "email.failed"
	EventEmailReplyReceived = "email.reply_received"
)

// sender implements the Sender interface.
type sender struct {
	svix svix.Client
}

// NewSender creates a new webhook Sender.
func NewSender(svixClient svix.Client) Sender {
	return &sender{svix: svixClient}
}

// SendEmailSent sends an email.sent webhook event.
func (s *sender) SendEmailSent(ctx context.Context, userID string, event *emailapiv1.EmailSentEvent) {
	s.send(ctx, userID, EventEmailSent, event)
}

// SendEmailDelivered sends an email.delivered webhook event.
func (s *sender) SendEmailDelivered(ctx context.Context, userID string, event *emailapiv1.EmailDeliveredEvent) {
	s.send(ctx, userID, EventEmailDelivered, event)
}

// SendEmailBounced sends an email.bounced webhook event.
func (s *sender) SendEmailBounced(ctx context.Context, userID string, event *emailapiv1.EmailBouncedEvent) {
	s.send(ctx, userID, EventEmailBounced, event)
}

// SendEmailComplained sends an email.complained webhook event.
func (s *sender) SendEmailComplained(ctx context.Context, userID string, event *emailapiv1.EmailComplainedEvent) {
	s.send(ctx, userID, EventEmailComplained, event)
}

// SendEmailRejected sends an email.rejected webhook event.
func (s *sender) SendEmailRejected(ctx context.Context, userID string, event *emailapiv1.EmailRejectedEvent) {
	s.send(ctx, userID, EventEmailRejected, event)
}

// SendEmailDelayed sends an email.delayed webhook event.
func (s *sender) SendEmailDelayed(ctx context.Context, userID string, event *emailapiv1.EmailDelayedEvent) {
	s.send(ctx, userID, EventEmailDelayed, event)
}

// SendEmailFailed sends an email.failed webhook event.
func (s *sender) SendEmailFailed(ctx context.Context, userID string, event *emailapiv1.EmailFailedEvent) {
	s.send(ctx, userID, EventEmailFailed, event)
}

// SendEmailReplyReceived sends an email.reply_received webhook event.
func (s *sender) SendEmailReplyReceived(ctx context.Context, userID string, event *emailapiv1.EmailReplyReceivedEvent) {
	s.send(ctx, userID, EventEmailReplyReceived, event)
}

// send handles the common webhook sending logic.
// It's fire-and-forget to avoid blocking the caller.
func (s *sender) send(ctx context.Context, userID, eventType string, payload proto.Message) {
	if s.svix == nil {
		return
	}

	go func() {
		// Use background context since this is fire-and-forget
		bgCtx := context.Background()

		// Ensure app exists for the user
		if err := s.svix.EnsureApp(bgCtx, userID, "User "+userID); err != nil {
			fmt.Printf("Warning: failed to ensure svix app for webhook: %v\n", err)
			return
		}

		// Build the webhook envelope
		envelope := &emailapiv1.WebhookEvent{
			EventType: eventType,
			Timestamp: timestamppb.New(time.Now().UTC()),
		}

		// Set the appropriate payload field based on event type
		switch p := payload.(type) {
		case *emailapiv1.EmailSentEvent:
			envelope.Payload = &emailapiv1.WebhookEvent_EmailSent{EmailSent: p}
		case *emailapiv1.EmailDeliveredEvent:
			envelope.Payload = &emailapiv1.WebhookEvent_EmailDelivered{EmailDelivered: p}
		case *emailapiv1.EmailBouncedEvent:
			envelope.Payload = &emailapiv1.WebhookEvent_EmailBounced{EmailBounced: p}
		case *emailapiv1.EmailComplainedEvent:
			envelope.Payload = &emailapiv1.WebhookEvent_EmailComplained{EmailComplained: p}
		case *emailapiv1.EmailRejectedEvent:
			envelope.Payload = &emailapiv1.WebhookEvent_EmailRejected{EmailRejected: p}
		case *emailapiv1.EmailDelayedEvent:
			envelope.Payload = &emailapiv1.WebhookEvent_EmailDelayed{EmailDelayed: p}
		case *emailapiv1.EmailFailedEvent:
			envelope.Payload = &emailapiv1.WebhookEvent_EmailFailed{EmailFailed: p}
		case *emailapiv1.EmailReplyReceivedEvent:
			envelope.Payload = &emailapiv1.WebhookEvent_EmailReplyReceived{EmailReplyReceived: p}
		}

		// Convert proto to JSON-friendly map for Svix
		jsonBytes, err := protojson.Marshal(envelope)
		if err != nil {
			fmt.Printf("Warning: failed to marshal webhook payload: %v\n", err)
			return
		}

		// Parse JSON to map for Svix
		var payloadMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &payloadMap); err != nil {
			fmt.Printf("Warning: failed to unmarshal webhook payload: %v\n", err)
			return
		}

		if err := s.svix.SendMessage(bgCtx, userID, eventType, payloadMap); err != nil {
			fmt.Printf("Warning: failed to send webhook for user %s: %v\n", userID, err)
		}
	}()
}
