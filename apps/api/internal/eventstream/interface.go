package eventstream

//go:generate mockgen -destination=mocks/mock_eventstream.go -package=mocks github.com/emailapi/api/internal/eventstream Publisher,Consumer

import (
	"context"

	v1 "github.com/emailapi/api/gen/v1"
)

// Publisher publishes events to the stream.
type Publisher interface {
	// Publish publishes an event to the user's stream.
	Publish(ctx context.Context, userID string, event *v1.Event) error
}

// Consumer consumes events from a stream using Redis Consumer Groups.
type Consumer interface {
	// Subscribe returns a channel of events for the user.
	// Uses Redis Consumer Groups with apiKeyID as the consumer identifier.
	// Unacknowledged events are automatically replayed on reconnect.
	// The channel is closed when ctx is cancelled or an error occurs.
	Subscribe(ctx context.Context, userID, apiKeyID string, eventTypes []v1.EventType, batchSize int32) (<-chan *v1.Event, error)

	// Ack acknowledges events as processed, removing them from the pending list.
	// This prevents the events from being replayed on reconnect.
	Ack(ctx context.Context, userID string, eventIDs []string) (int64, error)
}
