package limit

import (
	"context"
	"time"
)

// RateLimiterInterface for rate limit checking.
type RateLimiterInterface interface {
	Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error)
}

// RateLimitResult from the rate limiter.
type RateLimitResult struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
}

// ReputationCacheInterface for suspension/flag checking.
type ReputationCacheInterface interface {
	Get(ctx context.Context, userID string) (*ReputationStatus, error)
}

// ReputationStatus represents cached reputation state.
type ReputationStatus struct {
	IsSoftSuspended bool
	IsHardSuspended bool
	IsFlagged       bool // Deprecated: use IsSoftSuspended
	Score           float64
}

// UsageCacheInterface for usage tracking.
type UsageCacheInterface interface {
	GetDailyUsage(ctx context.Context, userID string) (int64, error)
	IncrementDailyUsage(ctx context.Context, userID string, count int64) error
}

// CreditCacheInterface for Polar credit state.
type CreditCacheInterface interface {
	GetState(ctx context.Context, userID string) (*CreditState, error)
}

// CreditState represents Polar credit state.
type CreditState struct {
	IsPaid       bool
	PlanType     string
	PolarBalance int64 // Monthly credit balance from Polar
}
