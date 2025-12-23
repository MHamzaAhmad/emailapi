package eventstream

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/emailapi/api/gen/v1"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

const (
	streamPrefix = "events:" // events:{userID}
	maxStreamLen = 10000     // Max events per user stream (~approximate)
)

// redisPublisher implements Publisher using Redis Streams.
type redisPublisher struct {
	client *redisrepo.Client
}

// NewPublisher creates a new event stream publisher.
func NewPublisher(client *redisrepo.Client) Publisher {
	return &redisPublisher{client: client}
}

// Publish publishes an event to the user's stream.
func (p *redisPublisher) Publish(ctx context.Context, userID string, event *v1.Event) error {
	stream := streamPrefix + userID

	// Set timestamp if not already set
	if event.Timestamp == nil {
		event.Timestamp = timestamppb.Now()
	}

	// Serialize event to JSON using protojson for proper enum handling
	data, err := protojson.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Add to stream with auto-generated ID (timestamp-based)
	id, err := p.client.XAdd(ctx, stream, map[string]interface{}{
		"data": string(data),
		"type": event.Type.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to add to stream: %w", err)
	}

	// Update event ID with Redis stream ID for cursor tracking
	event.Id = id

	// Trim stream to max length (approximate, efficient)
	_ = p.client.XTrimMaxLenApprox(ctx, stream, maxStreamLen)

	return nil
}
