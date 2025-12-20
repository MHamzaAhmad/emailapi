package repository

import (
	"context"

	"github.com/emailapi/api/internal/domain"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	// Create stores a new user.
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by their ID.
	GetByID(ctx context.Context, id string) (*domain.User, error)

	// GetByEmail retrieves a user by their email.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update updates an existing user.
	Update(ctx context.Context, user *domain.User) error

	// Delete removes a user.
	Delete(ctx context.Context, id string) error

	// List retrieves all users with pagination.
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
}

// APIKeyRepository defines the interface for API key data access.
type APIKeyRepository interface {
	// Create stores a new API key.
	Create(ctx context.Context, apiKey *domain.APIKey) error

	// GetByID retrieves an API key by its ID.
	GetByID(ctx context.Context, id string) (*domain.APIKey, error)

	// GetByHash retrieves an active API key by its hash.
	GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error)

	// GetByPrefix retrieves an API key by its prefix.
	GetByPrefix(ctx context.Context, keyPrefix string) (*domain.APIKey, error)

	// ListByUserID retrieves all API keys for a user.
	ListByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error)

	// Update updates an existing API key.
	Update(ctx context.Context, apiKey *domain.APIKey) error

	// UpdateLastUsed updates the last_used_at timestamp.
	UpdateLastUsed(ctx context.Context, id string) error

	// Revoke deactivates an API key.
	Revoke(ctx context.Context, id string) (*domain.APIKey, error)

	// Delete removes an API key.
	Delete(ctx context.Context, id string) error

	// CountActiveByUserID counts active API keys for a user.
	CountActiveByUserID(ctx context.Context, userID string) (int64, error)
}

// DomainRepository defines the interface for sending domain data access.
type DomainRepository interface {
	// Create stores a new sending domain.
	Create(ctx context.Context, d *domain.SendingDomain) error

	// GetByID retrieves a domain by its ID.
	GetByID(ctx context.Context, id string) (*domain.SendingDomain, error)

	// GetByDomainName retrieves a domain by its name for a specific user.
	GetByDomainName(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error)

	// GetByUserID retrieves all domains for a user.
	GetByUserID(ctx context.Context, userID string) ([]*domain.SendingDomain, error)

	// Update updates an existing domain.
	Update(ctx context.Context, d *domain.SendingDomain) error

	// Delete removes a domain.
	Delete(ctx context.Context, id string) error
}

// TODO: EmailRepository (to be reimplemented later)
// TODO: WebhookRepository (to be reimplemented later)
// TODO: LogRepository (to be reimplemented later)
