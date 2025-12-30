package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/repository"
)

// Store implements the service.Store interface using PostgreSQL.
// It aggregates all repository implementations.
type Store struct {
	pool    *pgxpool.Pool
	queries *db.Queries

	user        *UserRepository
	apiKey      *APIKeyRepository
	domain      *DomainRepository
	reputation  *ReputationRepository
	unsubscribe *UnsubscribeRepository
}

// StoreConfig holds configuration for the PostgreSQL store.
type StoreConfig struct {
	DatabaseURL     string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// NewStore creates a new PostgreSQL store with explicit pool configuration.
func NewStore(cfg StoreConfig) (*Store, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{
		pool:    pool,
		queries: db.New(pool),
	}
	store.user = NewUserRepository(pool)
	store.apiKey = NewAPIKeyRepository(pool)
	store.domain = NewDomainRepository(store.queries)
	store.reputation = NewReputationRepository(pool)
	store.unsubscribe = NewUnsubscribeRepository(pool)

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

// Reputation returns the reputation repository.
func (s *Store) Reputation() repository.ReputationRepository {
	return s.reputation
}

// Pool returns the underlying connection pool for use by other repositories.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// Queries returns the sqlc generated queries.
func (s *Store) Queries() *db.Queries {
	return s.queries
}

// Unsubscribe returns the unsubscribe repository.
func (s *Store) Unsubscribe() repository.UnsubscribeRepository {
	return s.unsubscribe
}
