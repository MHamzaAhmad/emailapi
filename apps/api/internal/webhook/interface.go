package webhook

//go:generate mockgen -destination=mocks/mock_webhook.go -package=mocks github.com/emailapi/api/internal/webhook Sender

import (
	"context"

	v1 "github.com/emailapi/api/gen/v1"
)

// Sender defines the interface for sending webhook events.
// Use this interface in services and workers for testability.
type Sender interface {
	// SendEmailSent notifies when an email is accepted for delivery.
	SendEmailSent(ctx context.Context, userID string, event *v1.EmailSentEvent)

	// SendEmailDelivered notifies when an email is delivered.
	SendEmailDelivered(ctx context.Context, userID string, event *v1.EmailDeliveredEvent)

	// SendEmailBounced notifies when an email bounces.
	SendEmailBounced(ctx context.Context, userID string, event *v1.EmailBouncedEvent)

	// SendEmailComplained notifies when a recipient marks email as spam.
	SendEmailComplained(ctx context.Context, userID string, event *v1.EmailComplainedEvent)

	// SendEmailRejected notifies when SES rejects an email.
	SendEmailRejected(ctx context.Context, userID string, event *v1.EmailRejectedEvent)

	// SendEmailDelayed notifies when email delivery is delayed.
	SendEmailDelayed(ctx context.Context, userID string, event *v1.EmailDelayedEvent)

	// SendEmailFailed notifies when email sending fails.
	SendEmailFailed(ctx context.Context, userID string, event *v1.EmailFailedEvent)

	// SendEmailReplied notifies when a reply is received.
	SendEmailReplied(ctx context.Context, userID string, event *v1.EmailRepliedEvent)
}
