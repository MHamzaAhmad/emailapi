package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Handler processes a specific type of event.
type Handler interface {
	// Handle processes the event and returns an error if processing failed.
	// Returning an error will cause the message to be retried.
	Handle(ctx context.Context, event *SESEvent) error
}

// InboundHandler processes inbound email events from S3.
// This is the new simplified interface - receives bucket and key directly.
type InboundHandler interface {
	HandleS3Event(ctx context.Context, bucket, key string) error
}

// Router routes incoming SQS messages to appropriate handlers.
type Router struct {
	handlers       map[EventType]Handler
	inboundHandler InboundHandler
	inboundBucket  string // Expected bucket name for inbound emails
}

// NewRouter creates a new event router.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[EventType]Handler),
	}
}

// RegisterHandler registers a handler for a specific event type.
func (r *Router) RegisterHandler(eventType EventType, handler Handler) {
	r.handlers[eventType] = handler
}

// RegisterInboundHandler registers the handler for inbound email events.
func (r *Router) RegisterInboundHandler(handler InboundHandler, bucketName string) {
	r.inboundHandler = handler
	r.inboundBucket = bucketName
}

// Route parses and routes the message to the appropriate handler.
func (r *Router) Route(ctx context.Context, body string) error {
	// Try EventBridge envelope first
	var envelope EventEnvelope
	if err := json.Unmarshal([]byte(body), &envelope); err == nil && envelope.Source != "" {
		return r.routeEventBridge(ctx, &envelope, body)
	}

	// Try SNS wrapper (legacy)
	var snsNotification SNSNotification
	if err := json.Unmarshal([]byte(body), &snsNotification); err == nil && snsNotification.Type != "" {
		body = snsNotification.Message
	}

	// Check for inbound email notification (legacy SNS format)
	if strings.Contains(body, `"notificationType":"Received"`) || strings.Contains(body, `"receipt":{`) {
		return r.routeLegacyInbound(ctx, body)
	}

	// Parse as SES sending event
	return r.routeSending(ctx, body)
}

// routeEventBridge handles EventBridge events.
func (r *Router) routeEventBridge(ctx context.Context, envelope *EventEnvelope, rawBody string) error {
	switch envelope.Source {
	case "aws.s3":
		// S3 event - likely inbound email
		return r.routeS3Event(ctx, envelope)
	case "aws.ses":
		// SES event - bounce, complaint, delivery, etc.
		detailBytes, _ := json.Marshal(envelope.Detail)
		return r.routeSending(ctx, string(detailBytes))
	default:
		fmt.Printf("Unknown EventBridge source: %s\n", envelope.Source)
		return nil
	}
}

// routeS3Event handles S3 object creation events from EventBridge.
func (r *Router) routeS3Event(ctx context.Context, envelope *EventEnvelope) error {
	detailBytes, err := json.Marshal(envelope.Detail)
	if err != nil {
		return fmt.Errorf("failed to marshal S3 event detail: %w", err)
	}

	var detail S3EventDetail
	if err := json.Unmarshal(detailBytes, &detail); err != nil {
		return fmt.Errorf("failed to parse S3 event detail: %w", err)
	}

	// Check if this is from our inbound email bucket
	if r.inboundHandler != nil && detail.Bucket.Name == r.inboundBucket {
		return r.inboundHandler.HandleS3Event(ctx, detail.Bucket.Name, detail.Object.Key)
	}

	fmt.Printf("S3 event from bucket %s not handled (expected %s)\n", detail.Bucket.Name, r.inboundBucket)
	return nil
}

// routeLegacyInbound handles legacy inbound email notifications (via SNS).
func (r *Router) routeLegacyInbound(ctx context.Context, body string) error {
	var event InboundEmailNotification
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return fmt.Errorf("failed to parse inbound email event: %w", err)
	}

	if r.inboundHandler == nil {
		return fmt.Errorf("no handler registered for inbound emails")
	}

	// Extract bucket and key from receipt action
	if event.Receipt.Action.Type == "S3" {
		return r.inboundHandler.HandleS3Event(ctx, event.Receipt.Action.BucketName, event.Receipt.Action.ObjectKey)
	}

	return fmt.Errorf("unsupported inbound action type: %s", event.Receipt.Action.Type)
}

// routeSending handles SES sending events (bounce, complaint, delivery, etc).
func (r *Router) routeSending(ctx context.Context, body string) error {
	var event SESEvent
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return fmt.Errorf("failed to parse SES event: %w", err)
	}

	eventType := EventType(event.EventType)
	handler, ok := r.handlers[eventType]
	if !ok {
		// No handler registered - log and skip (don't retry)
		fmt.Printf("No handler for event type: %s\n", eventType)
		return nil
	}

	return handler.Handle(ctx, &event)
}
