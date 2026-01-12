package limit

import (
	"context"

	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// ReputationCacheInterface for suspension/flag checking.
// Uses the redis cache interface directly.
type ReputationCacheInterface interface {
	Get(ctx context.Context, userID string) (*redisrepo.UserReputationStatus, error)
}

// CreditCacheInterface for Polar credit state and daily usage.
type CreditCacheInterface interface {
	GetState(ctx context.Context, userID string) (*redisrepo.CachedCustomerState, error)
	GetDailyUsage(ctx context.Context, userID string) (int64, error)
}

// SoftSuspendLimit is the daily email limit for soft suspended users (any plan).
const SoftSuspendLimit int64 = 10
