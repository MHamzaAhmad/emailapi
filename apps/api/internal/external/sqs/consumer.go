package sqs

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// MessageHandler processes SQS message bodies.
type MessageHandler interface {
	// Route processes the raw message body and returns an error if processing failed.
	Route(ctx context.Context, body string) error
}

// ConsumerConfig contains configuration for the SQS consumer.
type ConsumerConfig struct {
	// MaxMessages is the maximum number of messages to receive per poll (1-10).
	MaxMessages int
	// WaitTimeSeconds is the long-poll duration (1-20 seconds).
	WaitTimeSeconds int
	// InitialBackoff is the initial backoff duration on error.
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration.
	MaxBackoff time.Duration
}

// DefaultConsumerConfig returns sensible defaults for SQS consumption.
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		MaxMessages:     10,
		WaitTimeSeconds: 20, // Maximum SQS long-poll duration
		InitialBackoff:  5 * time.Second,
		MaxBackoff:      60 * time.Second,
	}
}

// Consumer continuously polls SQS and routes messages to a handler.
// It runs as a long-lived goroutine and should be started with `go consumer.Start(ctx)`.
type Consumer struct {
	client  Client
	handler MessageHandler
	config  ConsumerConfig
}

// NewConsumer creates a new SQS consumer.
func NewConsumer(client Client, handler MessageHandler, config ConsumerConfig) *Consumer {
	// Apply defaults for zero values
	if config.MaxMessages <= 0 || config.MaxMessages > 10 {
		config.MaxMessages = 10
	}
	if config.WaitTimeSeconds <= 0 || config.WaitTimeSeconds > 20 {
		config.WaitTimeSeconds = 20
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = 5 * time.Second
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = 60 * time.Second
	}

	return &Consumer{
		client:  client,
		handler: handler,
		config:  config,
	}
}

// Start begins the continuous polling loop.
// It blocks until the context is cancelled, returning nil on graceful shutdown.
func (c *Consumer) Start(ctx context.Context) error {
	backoff := c.config.InitialBackoff

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("SQS consumer shutting down")
			return nil
		default:
			// Continue polling
		}

		// Long-poll for messages
		messages, err := c.client.ReceiveMessages(ctx, c.config.MaxMessages, c.config.WaitTimeSeconds)
		if err != nil {
			// Check if context was cancelled during receive
			if ctx.Err() != nil {
				return nil
			}

			log.Error().Err(err).Dur("backoff", backoff).Msg("Failed to receive SQS messages, backing off")

			// Exponential backoff
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
				backoff = min(backoff*2, c.config.MaxBackoff)
				continue
			}
		}

		// Reset backoff on successful receive
		backoff = c.config.InitialBackoff

		if len(messages) == 0 {
			continue // No messages, poll again
		}

		log.Debug().Int("count", len(messages)).Msg("Received SQS messages")

		// Process each message
		for _, msg := range messages {
			c.processMessage(ctx, msg)
		}
	}
}

// processMessage handles a single message, logging errors but not failing the consumer.
func (c *Consumer) processMessage(ctx context.Context, msg Message) {
	start := time.Now()

	if err := c.handler.Route(ctx, msg.Body); err != nil {
		log.Error().
			Err(err).
			Str("message_id", msg.MessageID).
			Msg("Failed to process SQS message")
		// Don't delete - message will become visible again after visibility timeout
		return
	}

	// Successfully processed - delete from queue
	if err := c.client.DeleteMessage(ctx, msg.ReceiptHandle); err != nil {
		log.Warn().
			Err(err).
			Str("message_id", msg.MessageID).
			Msg("Failed to delete SQS message after processing")
		// Message was processed, but not deleted. It may be reprocessed,
		// but handlers should be idempotent.
		return
	}

	log.Debug().
		Str("message_id", msg.MessageID).
		Dur("duration", time.Since(start)).
		Msg("Processed SQS message")
}
