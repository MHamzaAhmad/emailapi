package handlers

import (
	"context"
	"fmt"

	"github.com/emailapi/api/internal/external/s3"
)

// InboundEmailHandler processes inbound email events from S3.
type InboundEmailHandler struct {
	s3Factory s3.FactoryInterface
	processor InboundEmailProcessor
}

// InboundEmailProcessor processes raw inbound emails from S3.
type InboundEmailProcessor interface {
	ProcessRawEmail(ctx context.Context, bucket, key string) error
}

// NewInboundEmailHandler creates a new inbound email handler.
func NewInboundEmailHandler(s3Factory s3.FactoryInterface, processor InboundEmailProcessor) *InboundEmailHandler {
	return &InboundEmailHandler{
		s3Factory: s3Factory,
		processor: processor,
	}
}

// HandleS3Event processes an S3 object creation event (inbound email).
// Implements events.InboundHandler interface.
func (h *InboundEmailHandler) HandleS3Event(ctx context.Context, bucket, key string) error {
	if h.processor == nil {
		return fmt.Errorf("no processor configured for inbound emails")
	}

	fmt.Printf("Processing inbound email: s3://%s/%s\n", bucket, key)

	// Delegate to the processor service which handles:
	// - Downloading from S3
	// - Parsing email headers (including X-SES-Virus-Verdict)
	// - Looking up routing
	// - Delivering webhook
	if err := h.processor.ProcessRawEmail(ctx, bucket, key); err != nil {
		return fmt.Errorf("failed to process inbound email: %w", err)
	}

	return nil
}
