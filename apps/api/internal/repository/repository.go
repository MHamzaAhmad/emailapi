package repository

import (
	"context"

	"github.com/emailapi/api/internal/domain"
)

// EmailRepository defines the interface for email data access.
type EmailRepository interface {
	// Create stores a new email.
	Create(ctx context.Context, email *domain.Email) error

	// GetByID retrieves an email by its ID.
	GetByID(ctx context.Context, id string) (*domain.Email, error)

	// GetByUserID retrieves all emails for a user.
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Email, error)

	// Update updates an existing email.
	Update(ctx context.Context, email *domain.Email) error

	// UpdateStatus updates only the status of an email.
	UpdateStatus(ctx context.Context, id string, status domain.EmailStatus) error
}

// LogRepository defines the interface for log data access.
// Uses ClickHouse for high-volume analytics.
type LogRepository interface {
	// Create stores a new log entry.
	Create(ctx context.Context, log *domain.Log) error

	// Query retrieves logs based on filters.
	Query(ctx context.Context, filter domain.LogFilter) ([]*domain.Log, error)

	// CountByEmailID counts logs for a specific email.
	CountByEmailID(ctx context.Context, emailID string) (int64, error)
}

// UserRepository defines the interface for user data access.
type UserRepository interface {
	// Create stores a new user.
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by their ID.
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by their email.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// GetByAPIKey retrieves a user by their API key.
	GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error)

	// Update updates an existing user.
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user.
	Delete(ctx context.Context, id string) error

	// UpdateAPIKey updates a user's API key.
	UpdateAPIKey(ctx context.Context, id string, hashedKey string, prefix string) error
}

// WebhookRepository defines the interface for webhook data access.
type WebhookRepository interface {
	// Create stores a new webhook.
	Create(ctx context.Context, webhook *domain.Webhook) error

	// GetByID retrieves a webhook by its ID.
	GetByID(ctx context.Context, id string) (*domain.Webhook, error)

	// GetByUserID retrieves all webhooks for a user.
	GetByUserID(ctx context.Context, userID string) ([]*domain.Webhook, error)

	// GetActiveByEvent retrieves all active webhooks for a specific event type.
	GetActiveByEvent(ctx context.Context, eventType domain.WebhookEventType) ([]*domain.Webhook, error)

	// Update updates an existing webhook.
	Update(ctx context.Context, webhook *domain.Webhook) error

	// Delete removes a webhook.
	Delete(ctx context.Context, id string) error

	// CreateDelivery stores a webhook delivery attempt.
	CreateDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error

	// GetDeliveriesByWebhookID retrieves delivery history for a webhook.
	GetDeliveriesByWebhookID(ctx context.Context, webhookID string, limit int) ([]*domain.WebhookDelivery, error)
}
