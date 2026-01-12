package limit

import (
	"context"
	"fmt"
)

// DailyQuotaChecker checks daily email limits.
// Free users: 100/day
// Soft suspended (any plan): 10/day
// Paid users (not soft suspended): unlimited
type DailyQuotaChecker struct {
	creditCache CreditCacheInterface
	repCache    ReputationCacheInterface
	freeLimit   int64 // Default: 100
}

// DailyQuotaConfig for the daily quota checker.
type DailyQuotaConfig struct {
	CreditCache CreditCacheInterface
	RepCache    ReputationCacheInterface
	FreeLimit   int64 // Default: 100
}

// NewDailyQuotaChecker creates a new daily quota checker.
func NewDailyQuotaChecker(cfg DailyQuotaConfig) *DailyQuotaChecker {
	limit := cfg.FreeLimit
	if limit <= 0 {
		limit = 100 // Default
	}
	return &DailyQuotaChecker{
		creditCache: cfg.CreditCache,
		repCache:    cfg.RepCache,
		freeLimit:   limit,
	}
}

// Name returns the checker name.
func (c *DailyQuotaChecker) Name() string { return "daily_quota" }

// Check verifies user is within daily limit.
func (c *DailyQuotaChecker) Check(ctx context.Context, userID string) (*CheckResult, error) {
	// Check soft suspension first
	isSoftSuspended := c.isSoftSuspended(ctx, userID)

	// Get plan status
	isPaid := c.isPaidUser(ctx, userID)

	// Determine daily limit
	var dailyLimit int64
	if isSoftSuspended {
		// Soft suspended = 10 emails/day for ANY plan
		dailyLimit = SoftSuspendLimit
	} else if isPaid {
		// Paid users (not soft suspended) = unlimited
		return Allowed().WithMeta("daily_limit", "-1"), nil
	} else {
		// Free users = configured limit (default 100)
		dailyLimit = c.freeLimit
	}

	// Get current usage
	usage, err := c.creditCache.GetDailyUsage(ctx, userID)
	if err != nil {
		// Fail open on cache error
		return Allowed(), nil
	}

	if usage >= dailyLimit {
		return Blocked(ReasonDailyExceeded, PriorityDailyQuota).
			WithMeta("daily_limit", fmt.Sprintf("%d", dailyLimit)).
			WithMeta("daily_usage", fmt.Sprintf("%d", usage)).
			WithMeta("soft_suspended", fmt.Sprintf("%t", isSoftSuspended)), nil
	}

	remaining := dailyLimit - usage
	return Allowed().
		WithMeta("daily_remaining", fmt.Sprintf("%d", remaining)).
		WithMeta("daily_limit", fmt.Sprintf("%d", dailyLimit)), nil
}

// isPaidUser checks if user has paid plan.
func (c *DailyQuotaChecker) isPaidUser(ctx context.Context, userID string) bool {
	if c.creditCache == nil {
		return false
	}
	state, err := c.creditCache.GetState(ctx, userID)
	if err != nil || state == nil {
		return false
	}
	return state.IsPaid
}

// isSoftSuspended checks if user is soft suspended.
func (c *DailyQuotaChecker) isSoftSuspended(ctx context.Context, userID string) bool {
	if c.repCache == nil {
		return false
	}
	status, err := c.repCache.Get(ctx, userID)
	if err != nil || status == nil {
		return false
	}
	return status.IsSoftSuspended || status.IsFlagged
}
