package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/eventstream"
	"github.com/emailapi/api/internal/external/svix"
)

// Event type constants matching Svix event types.
const (
	EventEmailSent       = "email.sent"
	EventEmailDelivered  = "email.delivered"
	EventEmailBounced    = "email.bounced"
	EventEmailComplained = "email.complained"
	EventEmailRejected   = "email.rejected"
	EventEmailDelayed    = "email.delayed"
	EventEmailFailed     = "email.failed"
	EventEmailReplied    = "email.replied"
)

// sender implements the Sender interface.
type sender struct {
	svix      svix.Client
	publisher eventstream.Publisher
}

// NewSender creates a new webhook Sender.
// publisher is optional - if nil, events won't be published to the stream.
func NewSender(svixClient svix.Client, publisher eventstream.Publisher) Sender {
	return &sender{svix: svixClient, publisher: publisher}
}

// SendEmailSent sends an email.sent webhook event.
func (s *sender) SendEmailSent(ctx context.Context, userID string, event *v1.EmailSentEvent) {
	s.send(ctx, userID, EventEmailSent, v1.EventType_EVENT_TYPE_EMAIL_SENT, event)
}

// SendEmailDelivered sends an email.delivered webhook event.
func (s *sender) SendEmailDelivered(ctx context.Context, userID string, event *v1.EmailDeliveredEvent) {
	s.send(ctx, userID, EventEmailDelivered, v1.EventType_EVENT_TYPE_EMAIL_DELIVERED, event)
}

// SendEmailBounced sends an email.bounced webhook event.
func (s *sender) SendEmailBounced(ctx context.Context, userID string, event *v1.EmailBouncedEvent) {
	s.send(ctx, userID, EventEmailBounced, v1.EventType_EVENT_TYPE_EMAIL_BOUNCED, event)
}

// SendEmailComplained sends an email.complained webhook event.
func (s *sender) SendEmailComplained(ctx context.Context, userID string, event *v1.EmailComplainedEvent) {
	s.send(ctx, userID, EventEmailComplained, v1.EventType_EVENT_TYPE_EMAIL_COMPLAINED, event)
}

// SendEmailRejected sends an email.rejected webhook event.
func (s *sender) SendEmailRejected(ctx context.Context, userID string, event *v1.EmailRejectedEvent) {
	s.send(ctx, userID, EventEmailRejected, v1.EventType_EVENT_TYPE_EMAIL_REJECTED, event)
}

// SendEmailDelayed sends an email.delayed webhook event.
func (s *sender) SendEmailDelayed(ctx context.Context, userID string, event *v1.EmailDelayedEvent) {
	s.send(ctx, userID, EventEmailDelayed, v1.EventType_EVENT_TYPE_EMAIL_DELAYED, event)
}

// SendEmailFailed sends an email.failed webhook event.
func (s *sender) SendEmailFailed(ctx context.Context, userID string, event *v1.EmailFailedEvent) {
	s.send(ctx, userID, EventEmailFailed, v1.EventType_EVENT_TYPE_EMAIL_FAILED, event)
}

// SendEmailReplied sends an email.replied webhook event.
func (s *sender) SendEmailReplied(ctx context.Context, userID string, event *v1.EmailRepliedEvent) {
	s.send(ctx, userID, EventEmailReplied, v1.EventType_EVENT_TYPE_EMAIL_REPLIED, event)
}

// send handles the common webhook sending logic.
// It's fire-and-forget to avoid blocking the caller.
func (s *sender) send(ctx context.Context, userID, eventTypeStr string, eventType v1.EventType, payload proto.Message) {
	// Build the event for both webhook and stream
	event := s.buildEvent(eventType, payload)

	// Publish to event stream (fire-and-forget)
	if s.publisher != nil {
		go func() {
			if err := s.publisher.Publish(context.Background(), userID, event); err != nil {
				fmt.Printf("Warning: failed to publish to event stream: %v\n", err)
			}
		}()
	}

	// Send webhook via Svix
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

		// Build the webhook envelope using the new Event type
		envelope := &v1.WebhookEvent{
			Event: event,
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

		if err := s.svix.SendMessage(bgCtx, userID, eventTypeStr, payloadMap); err != nil {
			fmt.Printf("Warning: failed to send webhook for user %s: %v\n", userID, err)
		}
	}()
}

// buildEvent creates an Event from the payload.
func (s *sender) buildEvent(eventType v1.EventType, payload proto.Message) *v1.Event {
	event := &v1.Event{
		Type:      eventType,
		Timestamp: timestamppb.New(time.Now().UTC()),
	}

	// Set the appropriate payload field based on type
	switch p := payload.(type) {
	case *v1.EmailSentEvent:
		event.Payload = &v1.Event_EmailSent{EmailSent: p}
	case *v1.EmailDeliveredEvent:
		event.Payload = &v1.Event_EmailDelivered{EmailDelivered: p}
	case *v1.EmailBouncedEvent:
		event.Payload = &v1.Event_EmailBounced{EmailBounced: p}
	case *v1.EmailComplainedEvent:
		event.Payload = &v1.Event_EmailComplained{EmailComplained: p}
	case *v1.EmailRejectedEvent:
		event.Payload = &v1.Event_EmailRejected{EmailRejected: p}
	case *v1.EmailDelayedEvent:
		event.Payload = &v1.Event_EmailDelayed{EmailDelayed: p}
	case *v1.EmailFailedEvent:
		event.Payload = &v1.Event_EmailFailed{EmailFailed: p}
	case *v1.EmailRepliedEvent:
		event.Payload = &v1.Event_EmailReplied{EmailReplied: p}
	}

	return event
}
