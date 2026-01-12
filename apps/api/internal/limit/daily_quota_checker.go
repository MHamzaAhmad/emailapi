package limit

import (
	"context"
	"fmt"
)

// DailyQuotaChecker checks daily email limits.
// Free users: 100/day (10 if soft suspended)
// Paid users: unlimited
type DailyQuotaChecker struct {
	usageCache  UsageCacheInterface
	repCache    ReputationCacheInterface
	creditCache CreditCacheInterface
	freeLimit   int64 // Default: 100
}

// DailyQuotaConfig for the daily quota checker.
type DailyQuotaConfig struct {
	UsageCache  UsageCacheInterface
	RepCache    ReputationCacheInterface
	CreditCache CreditCacheInterface
	FreeLimit   int64 // e.g., 100
}

// NewDailyQuotaChecker creates a new daily quota checker.
func NewDailyQuotaChecker(cfg DailyQuotaConfig) *DailyQuotaChecker {
	limit := cfg.FreeLimit
	if limit <= 0 {
		limit = 100 // Default
	}
	return &DailyQuotaChecker{
		usageCache:  cfg.UsageCache,
		repCache:    cfg.RepCache,
		creditCache: cfg.CreditCache,
		freeLimit:   limit,
	}
}

// Name returns the checker name.
func (c *DailyQuotaChecker) Name() string { return "daily_quota" }

// Check verifies user is within daily limit.
func (c *DailyQuotaChecker) Check(ctx context.Context, userID string) (*CheckResult, error) {
	// Get plan status
	isPaid := c.isPaidUser(ctx, userID)

	// Paid users have no daily limit
	if isPaid {
		return Allowed().WithMeta("daily_limit", "-1"), nil
	}

	// Get effective limit (10% for soft suspended)
	effectiveLimit := c.getEffectiveLimit(ctx, userID)

	// Get current usage
	usage, err := c.usageCache.GetDailyUsage(ctx, userID)
	if err != nil {
		// Fail open on cache error
		return Allowed(), nil
	}

	if usage >= effectiveLimit {
		return Blocked(ReasonDailyExceeded, PriorityDailyQuota).
			WithMeta("daily_limit", fmt.Sprintf("%d", effectiveLimit)).
			WithMeta("daily_usage", fmt.Sprintf("%d", usage)), nil
	}

	remaining := effectiveLimit - usage
	return Allowed().
		WithMeta("daily_remaining", fmt.Sprintf("%d", remaining)).
		WithMeta("daily_limit", fmt.Sprintf("%d", effectiveLimit)), nil
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

// getEffectiveLimit returns daily limit, reduced for soft suspended.
func (c *DailyQuotaChecker) getEffectiveLimit(ctx context.Context, userID string) int64 {
	if c.repCache == nil {
		return c.freeLimit
	}

	status, err := c.repCache.Get(ctx, userID)
	if err != nil || status == nil {
		return c.freeLimit
	}

	// Soft suspended users get 10% of limit
	if status.IsSoftSuspended {
		reduced := c.freeLimit / 10
		if reduced < 1 {
			reduced = 1
		}
		return reduced
	}

	return c.freeLimit
}
