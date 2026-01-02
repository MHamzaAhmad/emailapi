package suppression

//go:generate mockgen -destination=mocks/mock_suppression.go -package=mocks github.com/emailapi/api/internal/repository/suppression RepositoryInterface

import (
	"context"
)

// RepositoryInterface defines the interface for suppression list operations.
// This hybrid repository uses Redis for fast reads and PostgreSQL for durability.
type RepositoryInterface interface {
	// Add stores a suppression entry in both Redis and PostgreSQL.
	Add(ctx context.Context, entry *Entry) error

	// CheckBatch checks multiple email hashes for suppression using Redis MGET.
	// Returns the list of hashes that are suppressed.
	CheckBatch(ctx context.Context, hashes []string) ([]string, error)

	// Remove deletes a suppression entry from both Redis and PostgreSQL.
	Remove(ctx context.Context, emailHash string) error

	// SyncFromPostgres reloads all active suppressions from PostgreSQL into Redis.
	// This is called on startup to recover from Redis restarts.
	SyncFromPostgres(ctx context.Context) error

	// Cleanup deletes expired entries from PostgreSQL.
	// Redis handles its own TTL expiration automatically.
	Cleanup(ctx context.Context) (int64, error)
}

// Ensure concrete type implements interface
var _ RepositoryInterface = (*Repository)(nil)
