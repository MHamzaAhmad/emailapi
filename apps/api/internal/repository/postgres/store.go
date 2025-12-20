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

	email   *EmailRepository
	user    *UserRepository
	webhook *WebhookRepository
	domain  *DomainRepository
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
	store.email = &EmailRepository{pool: pool}
	store.user = &UserRepository{pool: pool}
	store.webhook = &WebhookRepository{pool: pool}
	store.domain = &DomainRepository{pool: pool}

	return store, nil
}

// Close closes the database connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Emails returns the email repository.
func (s *Store) Emails() repository.EmailRepository {
	return s.email
}

// Users returns the user repository.
func (s *Store) Users() repository.UserRepository {
	return s.user
}

// Webhooks returns the webhook repository.
func (s *Store) Webhooks() repository.WebhookRepository {
	return s.webhook
}

// Domains returns the domain repository.
func (s *Store) Domains() repository.DomainRepository {
	return s.domain
}
