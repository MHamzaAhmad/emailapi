package eventstream

import (
	"context"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/emailapi/api/gen/v1"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// redisConsumer implements Consumer using Redis Streams.
type redisConsumer struct {
	client *redisrepo.Client
}

// NewConsumer creates a new event stream consumer.
func NewConsumer(client *redisrepo.Client) Consumer {
	return &redisConsumer{client: client}
}

// Subscribe returns a channel of events starting from cursor.
// The channel is closed when ctx is cancelled or an error occurs.
func (c *redisConsumer) Subscribe(ctx context.Context, userID, cursor string, eventTypes []v1.EventType, batchSize int32) (<-chan *v1.Event, error) {
	stream := streamPrefix + userID

	// Default cursor to latest if not specified
	if cursor == "" {
		cursor = "$"
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

		lastID := cursor
		eventTypeSet := make(map[v1.EventType]bool)
		for _, t := range eventTypes {
			eventTypeSet[t] = true
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Block for up to 5 seconds waiting for new events
			// streams format: [stream1, stream2, ..., id1, id2, ...]
			streams, err := c.client.XRead(ctx, []string{stream, lastID}, int64(batchSize), 5*time.Second)
			if err != nil {
				// Timeout or redis.Nil is expected, continue
				// Check if context was cancelled
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}

			for _, s := range streams {
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
						lastID = msg.ID
						continue
					}

					select {
					case eventCh <- &event:
						lastID = msg.ID
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return eventCh, nil
}
