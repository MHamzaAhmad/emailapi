package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emailapi/api/internal/repository"
)

// Store implements the service.Store interface using PostgreSQL.
// It aggregates all repository implementations.
type Store struct {
	pool *pgxpool.Pool

	user   *UserRepository
	apiKey *APIKeyRepository
	domain *DomainRepository
}

// NewStore creates a new PostgreSQL store.
func NewStore(databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{pool: pool}
	store.user = NewUserRepository(pool)
	store.apiKey = NewAPIKeyRepository(pool)
	store.domain = NewDomainRepository(pool)

	return store, nil
}

// Close closes the database connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Users returns the user repository.
func (s *Store) Users() repository.UserRepository {
	return s.user
}

// APIKeys returns the API key repository.
func (s *Store) APIKeys() repository.APIKeyRepository {
	return s.apiKey
}

// Domains returns the domain repository.
func (s *Store) Domains() repository.DomainRepository {
	return s.domain
}

// Pool returns the underlying connection pool for use by other repositories.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}
