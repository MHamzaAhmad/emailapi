package sqs

//go:generate mockgen -destination=mocks/mock_sqs.go -package=mocks github.com/emailapi/api/internal/external/sqs Client

import "context"

// Message represents an SQS message.
type Message struct {
	MessageID     string
	ReceiptHandle string
	Body          string
	// MessageGroupID is set for FIFO queues
	MessageGroupID string
	// Attributes contains message attributes (e.g., ApproximateReceiveCount)
	Attributes map[string]string
}

// Client defines the interface for AWS SQS operations.
type Client interface {
	// ReceiveMessages long-polls for up to maxMessages from the queue.
	// Uses waitTimeSeconds for long polling (typically 20 seconds).
	ReceiveMessages(ctx context.Context, maxMessages int, waitTimeSeconds int) ([]Message, error)

	// DeleteMessage removes a message from the queue after successful processing.
	DeleteMessage(ctx context.Context, receiptHandle string) error

	// ChangeMessageVisibility extends the visibility timeout for a message.
	// Use this when processing takes longer than expected.
	ChangeMessageVisibility(ctx context.Context, receiptHandle string, visibilityTimeoutSeconds int) error
}
