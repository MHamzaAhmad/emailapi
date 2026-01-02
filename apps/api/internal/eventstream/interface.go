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

// Consumer consumes events from a stream.
type Consumer interface {
	// Subscribe returns a channel of events starting from cursor.
	// The channel is closed when ctx is cancelled or an error occurs.
	Subscribe(ctx context.Context, userID, cursor string, eventTypes []v1.EventType, batchSize int32) (<-chan *v1.Event, error)
}
