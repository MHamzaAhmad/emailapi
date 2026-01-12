package interceptor

//go:generate mockgen -destination=mocks/mock_interceptor.go -package=mocks github.com/emailapi/api/internal/transport/connect/interceptor APIKeyValidator,RateLimiterInterface,CreditCacheInterface,PolarClientInterface,UserRepositoryInterface,UserLookup,UserRoleLookup,ReputationCacheInterface

import (
	"context"
	"time"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/polar"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// APIKeyValidator validates API keys and returns user/key info.
type APIKeyValidator interface {
	ValidateAndGetUser(ctx context.Context, rawKey string) (*domain.User, *domain.APIKey, error)
}

// RateLimiterInterface provides rate limiting operations.
type RateLimiterInterface interface {
	Check(ctx context.Context, key string, limit int, window time.Duration) (*redisrepo.RateLimitResult, error)
	IncrementStreams(ctx context.Context, userID string, maxStreams int) (bool, error)
	DecrementStreams(ctx context.Context, userID string) error
}

// CreditCacheInterface - reuse from redis package
type CreditCacheInterface = redisrepo.CreditCacheInterface

// ReputationCacheInterface - reuse from redis package
type ReputationCacheInterface = redisrepo.ReputationCacheInterface

// PolarClientInterface - reuse from polar package
type PolarClientInterface = polar.Client

// UserRepositoryInterface for usage interceptor provisioning
type UserRepositoryInterface interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	UpdatePolarCustomerID(ctx context.Context, id, polarCustomerID string) error
}
