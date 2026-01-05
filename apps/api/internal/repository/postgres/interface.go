package postgres

//go:generate mockgen -destination=mocks/mock_postgres.go -package=mocks github.com/emailapi/api/internal/repository/postgres UserRepository,APIKeyRepository,DomainRepository,EmailRepository,ReputationRepository,UnsubscribeRepository

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

	// GetByExternalID retrieves a user by their Clerk external ID.
	GetByExternalID(ctx context.Context, externalID string) (*domain.User, error)

	// GetByPolarCustomerID retrieves a user by their Polar customer ID.
	GetByPolarCustomerID(ctx context.Context, polarCustomerID string) (*domain.User, error)

	// Update updates an existing user.
	Update(ctx context.Context, user *domain.User) error

	// UpdatePlan updates only the user's plan.
	UpdatePlan(ctx context.Context, id string, plan domain.UserPlan) error

	// UpdatePolarCustomerID updates only the user's Polar customer ID.
	UpdatePolarCustomerID(ctx context.Context, id, polarCustomerID string) error

	// Delete removes a user.
	Delete(ctx context.Context, id string) error

	// List retrieves all users with pagination.
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)

	// Count returns total number of users.
	Count(ctx context.Context) (int, error)

	// ListWithReputation retrieves users with their suspension/flag status.
	ListWithReputation(ctx context.Context, limit, offset int) ([]*domain.AdminUser, error)

	// GetWithReputation retrieves a user with full reputation data.
	GetWithReputation(ctx context.Context, userID string) (*domain.UserWithReputation, error)
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

	// ListByUserID retrieves all API keys for a user with pagination.
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.APIKey, error)

	// CountByUserID returns the total number of API keys for a user.
	CountByUserID(ctx context.Context, userID string) (int, error)

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

	// GetByUserID retrieves domains for a user with pagination.
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.SendingDomain, error)

	// CountByUserID returns the total number of domains for a user.
	CountByUserID(ctx context.Context, userID string) (int, error)

	// Update updates an existing domain.
	Update(ctx context.Context, d *domain.SendingDomain) error

	// Delete removes a domain.
	Delete(ctx context.Context, id string) error
}

// EmailRepository defines the interface for email data access.
type EmailRepository interface {
	// AddEmail stores a new email.
	AddEmail(ctx context.Context, email *domain.Email, eventType string) error
}

// ReputationRepository defines the interface for reputation data access.
type ReputationRepository interface {
	// EnsureExists creates a reputation record if it doesn't exist.
	EnsureExists(ctx context.Context, userID string) error

	// Get retrieves reputation stats for a user.
	Get(ctx context.Context, userID string) (*domain.UserReputation, error)

	// InsertIncident records a new reputation incident.
	InsertIncident(ctx context.Context, incident *domain.ReputationIncident) error

	// CountIncidents returns aggregated incident counts for a user.
	CountIncidents(ctx context.Context, userID string) (*domain.IncidentStats, error)

	// UpdateStats updates the reputation statistics.
	UpdateStats(ctx context.Context, userID string, stats *domain.UserReputation) error

	// Suspend marks a user as suspended.
	Suspend(ctx context.Context, userID, suspendedBy, reason string) error

	// Unsuspend removes suspension from a user.
	Unsuspend(ctx context.Context, userID string) error

	// ListFlagged lists flagged users with pagination.
	ListFlagged(ctx context.Context, limit, offset int) ([]*domain.UserReputation, int, error)

	// ListIncidents lists incidents for a user with pagination.
	ListIncidents(ctx context.Context, userID string, limit, offset int) ([]*domain.ReputationIncident, error)
}

// UnsubscribeRepository defines the interface for unsubscribe list data access.
type UnsubscribeRepository interface {
	// Add adds an email to the unsubscribe list for a user.
	Add(ctx context.Context, entry *domain.UnsubscribeEntry) error

	// GetByUserAndHash retrieves an unsubscribe entry by user and email hash.
	GetByUserAndHash(ctx context.Context, userID, emailHash string) (*domain.UnsubscribeEntry, error)

	// CheckBatch checks multiple email hashes for a user, returns unsubscribed hashes.
	CheckBatch(ctx context.Context, userID string, hashes []string) ([]string, error)

	// Delete removes an email from the unsubscribe list.
	Delete(ctx context.Context, userID, emailHash string) error

	// ListByUserID retrieves unsubscribes for a user with pagination.
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.UnsubscribeEntry, error)

	// CountByUserID returns total unsubscribes for a user.
	CountByUserID(ctx context.Context, userID string) (int64, error)

	// ListAll returns all unsubscribes (for cache sync).
	ListAll(ctx context.Context) ([]UnsubscribeItem, error)
}

// UnsubscribeItem represents an item returned from ListAll.
type UnsubscribeItem struct {
	UserID    string
	EmailHash string
}

// Ensure concrete types implement interfaces
var _ UserRepository = (*UserRepositoryImpl)(nil)
var _ APIKeyRepository = (*APIKeyRepositoryImpl)(nil)
var _ DomainRepository = (*DomainRepositoryImpl)(nil)
var _ ReputationRepository = (*ReputationRepositoryImpl)(nil)
var _ UnsubscribeRepository = (*UnsubscribeRepositoryImpl)(nil)
