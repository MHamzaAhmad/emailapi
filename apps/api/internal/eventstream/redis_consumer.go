package eventstream

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/emailapi/api/gen/v1"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// redisConsumer implements Consumer using Redis Streams with Consumer Groups.
// Uses API key ID as the consumer identifier for at-least-once delivery.
type redisConsumer struct {
	client *redisrepo.Client
}

// NewConsumer creates a new event stream consumer.
func NewConsumer(client *redisrepo.Client) Consumer {
	return &redisConsumer{client: client}
}

// Subscribe returns a channel of events using Redis Consumer Groups.
// The API key ID is used as the consumer identifier, allowing:
// - Automatic replay of unacknowledged events on reconnect
// - Load balancing across multiple consumers with the same API key
//
// Events must be acknowledged via Ack() to prevent replay.
// The channel is closed when ctx is cancelled or an error occurs.
func (c *redisConsumer) Subscribe(ctx context.Context, userID, apiKeyID string, eventTypes []v1.EventType, batchSize int32) (<-chan *v1.Event, error) {
	stream := streamPrefix + userID
	group := userID // User ID as group name (events per user)

	// Create consumer group if it doesn't exist
	// Start from "0" to ensure we don't miss any events
	if err := c.client.XGroupCreateMkStream(ctx, stream, group, "0"); err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	// Validate batch size
	if batchSize <= 0 {
		batchSize = 10
	}
	if batchSize > 100 {
		batchSize = 100
	}

	eventCh := make(chan *v1.Event, batchSize)

	go func() {
		defer close(eventCh)

		eventTypeSet := make(map[v1.EventType]bool)
		for _, t := range eventTypes {
			eventTypeSet[t] = true
		}

		// First, read pending (unacknowledged) events for this consumer
		// Use "0" to read from beginning of pending list
		pendingDone := false

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var id string
			if !pendingDone {
				// Read pending events first
				id = "0"
			} else {
				// Then read new events
				id = ">"
			}

			// Block for up to 5 seconds waiting for new events
			streams, err := c.client.XReadGroup(ctx, group, apiKeyID, stream, id, int64(batchSize), 5*time.Second)
			if err != nil {
				// Check if context was cancelled
				select {
				case <-ctx.Done():
					return
				default:
					// Timeout or redis.Nil is expected when no events
					if !pendingDone && id == "0" {
						// No pending events, switch to new events
						pendingDone = true
					}
					continue
				}
			}

			// If reading pending and got empty result, switch to new events
			if !pendingDone && len(streams) == 0 {
				pendingDone = true
				continue
			}

			if len(streams) == 0 && !pendingDone {
				pendingDone = true
				continue
			}

			for _, s := range streams {
				if len(s.Messages) == 0 && !pendingDone {
					pendingDone = true
					continue
				}

				for _, msg := range s.Messages {
					data, ok := msg.Values["data"].(string)
					if !ok {
						continue
					}

					var event v1.Event
					if err := protojson.Unmarshal([]byte(data), &event); err != nil {
						continue
					}

					// Set ID from Redis stream ID
					event.Id = msg.ID

					// Filter by event type if specified
					if len(eventTypeSet) > 0 && !eventTypeSet[event.Type] {
						// Still need to ack filtered events to prevent replay
						_, _ = c.client.XAck(ctx, stream, group, msg.ID)
						continue
					}

					select {
					case eventCh <- &event:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return eventCh, nil
}

// Ack acknowledges events as processed, removing them from the pending list.
// This prevents the events from being replayed on reconnect.
func (c *redisConsumer) Ack(ctx context.Context, userID string, eventIDs []string) (int64, error) {
	if len(eventIDs) == 0 {
		return 0, nil
	}
	stream := streamPrefix + userID
	group := userID
	return c.client.XAck(ctx, stream, group, eventIDs...)
}
