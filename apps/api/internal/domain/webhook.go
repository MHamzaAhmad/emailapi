package domain

import (
	"time"
)

// WebhookEventType represents the type of event that triggers a webhook.
type WebhookEventType string

const (
	WebhookEventEmailSent      WebhookEventType = "email.sent"
	WebhookEventEmailDelivered WebhookEventType = "email.delivered"
	WebhookEventEmailFailed    WebhookEventType = "email.failed"
	WebhookEventEmailBounced   WebhookEventType = "email.bounced"
	WebhookEventEmailOpened    WebhookEventType = "email.opened"
	WebhookEventEmailClicked   WebhookEventType = "email.clicked"
)

// Webhook represents a webhook configuration.
type Webhook struct {
	ID          string             `json:"id"`
	UserID      string             `json:"user_id"`
	Name        string             `json:"name"`
	URL         string             `json:"url"`
	Secret      string             `json:"-"` // Never expose in JSON
	Events      []WebhookEventType `json:"events"`
	IsActive    bool               `json:"is_active"`
	RetryCount  int                `json:"retry_count"`
	LastSuccess *time.Time         `json:"last_success,omitempty"`
	LastFailure *time.Time         `json:"last_failure,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// CreateWebhookRequest represents a request to create a webhook.
type CreateWebhookRequest struct {
	Name   string             `json:"name" binding:"required"`
	URL    string             `json:"url" binding:"required,url"`
	Events []WebhookEventType `json:"events" binding:"required,min=1"`
}

// UpdateWebhookRequest represents a request to update a webhook.
type UpdateWebhookRequest struct {
	Name     *string             `json:"name,omitempty"`
	URL      *string             `json:"url,omitempty" binding:"omitempty,url"`
	Events   []WebhookEventType  `json:"events,omitempty"`
	IsActive *bool               `json:"is_active,omitempty"`
}

// WebhookDelivery represents a webhook delivery attempt.
type WebhookDelivery struct {
	ID           string           `json:"id"`
	WebhookID    string           `json:"webhook_id"`
	EventType    WebhookEventType `json:"event_type"`
	Payload      string           `json:"payload"`
	ResponseCode int              `json:"response_code"`
	ResponseBody string           `json:"response_body,omitempty"`
	Success      bool             `json:"success"`
	AttemptCount int              `json:"attempt_count"`
	NextRetry    *time.Time       `json:"next_retry,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
}
