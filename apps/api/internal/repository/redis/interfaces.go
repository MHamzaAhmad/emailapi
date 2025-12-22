package redis

//go:generate mockgen -destination=mocks/mock_cache.go -package=mocks github.com/emailapi/api/internal/repository/redis DomainCacheInterface,APIKeyCacheInterface,UserCacheInterface

import (
	"context"

	"github.com/emailapi/api/internal/domain"
)

// DomainCacheInterface defines the interface for domain caching operations.
type DomainCacheInterface interface {
	GetByID(ctx context.Context, id string) (*domain.SendingDomain, error)
	SetByID(ctx context.Context, d *domain.SendingDomain) error
	GetByUserID(ctx context.Context, userID string) ([]*domain.SendingDomain, error)
	SetByUserID(ctx context.Context, userID string, domains []*domain.SendingDomain) error
	InvalidateByID(ctx context.Context, id string) error
	InvalidateByUserID(ctx context.Context, userID string) error
	InvalidateAll(ctx context.Context, id, userID string) error
}

// APIKeyCacheInterface defines the interface for API key caching operations.
type APIKeyCacheInterface interface {
	GetByID(ctx context.Context, id string) (*domain.APIKey, error)
	SetByID(ctx context.Context, k *domain.APIKey) error
	GetByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error)
	SetByPrefix(ctx context.Context, k *domain.APIKey) error
	GetByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error)
	SetByUserID(ctx context.Context, userID string, keys []*domain.APIKey) error
	InvalidateByID(ctx context.Context, id string) error
	InvalidateByPrefix(ctx context.Context, prefix string) error
	InvalidateByUserID(ctx context.Context, userID string) error
	InvalidateAll(ctx context.Context, id, prefix, userID string) error
}

// UserCacheInterface defines the interface for user caching operations.
type UserCacheInterface interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	SetByID(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	SetByEmail(ctx context.Context, u *domain.User) error
	InvalidateByID(ctx context.Context, id string) error
	InvalidateByEmail(ctx context.Context, email string) error
	InvalidateAll(ctx context.Context, id, email string) error
}

// Ensure concrete types implement interfaces
var _ DomainCacheInterface = (*DomainCache)(nil)
var _ APIKeyCacheInterface = (*APIKeyCache)(nil)
var _ UserCacheInterface = (*UserCache)(nil)
