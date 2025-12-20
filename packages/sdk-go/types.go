package emailapi

import "time"

// EmailStatus represents the status of an email.
type EmailStatus string

const (
	EmailStatusPending   EmailStatus = "pending"
	EmailStatusSent      EmailStatus = "sent"
	EmailStatusDelivered EmailStatus = "delivered"
	EmailStatusFailed    EmailStatus = "failed"
	EmailStatusBounced   EmailStatus = "bounced"
)

// SendEmailRequest represents a request to send an email.
type SendEmailRequest struct {
	From        string                 `json:"from"`
	To          []string               `json:"to"`
	Cc          []string               `json:"cc,omitempty"`
	Bcc         []string               `json:"bcc,omitempty"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body,omitempty"`
	HTML        string                 `json:"html,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
}

// SendEmailResponse represents the response from sending an email.
type SendEmailResponse struct {
	ID     string      `json:"id"`
	Status EmailStatus `json:"status"`
}

// Email represents an email.
type Email struct {
	ID          string                 `json:"id"`
	From        string                 `json:"from"`
	To          []string               `json:"to"`
	Cc          []string               `json:"cc,omitempty"`
	Bcc         []string               `json:"bcc,omitempty"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body,omitempty"`
	HTML        string                 `json:"html,omitempty"`
	Status      EmailStatus            `json:"status"`
	ProviderID  string                 `json:"provider_id,omitempty"`
	UserID      string                 `json:"user_id"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	SentAt      *time.Time             `json:"sent_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// EmailListResponse represents a list of emails.
type EmailListResponse struct {
	Data   []Email `json:"data"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// WebhookEventType represents a webhook event type.
type WebhookEventType string

const (
	WebhookEventEmailSent      WebhookEventType = "email.sent"
	WebhookEventEmailDelivered WebhookEventType = "email.delivered"
	WebhookEventEmailFailed    WebhookEventType = "email.failed"
	WebhookEventEmailBounced   WebhookEventType = "email.bounced"
	WebhookEventEmailOpened    WebhookEventType = "email.opened"
	WebhookEventEmailClicked   WebhookEventType = "email.clicked"
)

// CreateWebhookRequest represents a request to create a webhook.
type CreateWebhookRequest struct {
	Name   string             `json:"name"`
	URL    string             `json:"url"`
	Events []WebhookEventType `json:"events"`
}

// UpdateWebhookRequest represents a request to update a webhook.
type UpdateWebhookRequest struct {
	Name     *string            `json:"name,omitempty"`
	URL      *string            `json:"url,omitempty"`
	Events   []WebhookEventType `json:"events,omitempty"`
	IsActive *bool              `json:"is_active,omitempty"`
}

// Webhook represents a webhook.
type Webhook struct {
	ID          string             `json:"id"`
	UserID      string             `json:"user_id"`
	Name        string             `json:"name"`
	URL         string             `json:"url"`
	Events      []WebhookEventType `json:"events"`
	IsActive    bool               `json:"is_active"`
	RetryCount  int                `json:"retry_count"`
	LastSuccess *time.Time         `json:"last_success,omitempty"`
	LastFailure *time.Time         `json:"last_failure,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// WebhookWithSecret represents a webhook with its secret.
type WebhookWithSecret struct {
	Webhook Webhook `json:"webhook"`
	Secret  string  `json:"secret"`
	Message string  `json:"message"`
}

// WebhookListResponse represents a list of webhooks.
type WebhookListResponse struct {
	Data []Webhook `json:"data"`
}

// UserRole represents a user's role.
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleMember UserRole = "member"
)

// User represents a user.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         UserRole  `json:"role"`
	APIKeyPrefix string    `json:"api_key_prefix,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// APIKeyResponse represents a response containing an API key.
type APIKeyResponse struct {
	APIKey  string `json:"api_key"`
	Message string `json:"message"`
}
