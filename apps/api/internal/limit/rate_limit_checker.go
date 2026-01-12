package limit

import (
	"context"
	"strconv"
	"time"
)

// RateLimitChecker checks API rate limits.
// Applies reduced limits for soft suspended users.
type RateLimitChecker struct {
	limiter   RateLimiterInterface
	repCache  ReputationCacheInterface
	baseLimit int           // Normal limit per window
	window    time.Duration // Rate limit window
}

// RateLimitConfig for the rate limit checker.
type RateLimitConfig struct {
	Limiter   RateLimiterInterface
	RepCache  ReputationCacheInterface
	BaseLimit int           // e.g., 100 requests/min
	Window    time.Duration // e.g., time.Minute
}

// NewRateLimitChecker creates a new rate limit checker.
func NewRateLimitChecker(cfg RateLimitConfig) *RateLimitChecker {
	return &RateLimitChecker{
		limiter:   cfg.Limiter,
		repCache:  cfg.RepCache,
		baseLimit: cfg.BaseLimit,
		window:    cfg.Window,
	}
}

// Name returns the checker name.
func (c *RateLimitChecker) Name() string { return "rate_limit" }

// Check performs rate limit check with effective limit.
func (c *RateLimitChecker) Check(ctx context.Context, userID string) (*CheckResult, error) {
	if c.limiter == nil {
		return Allowed(), nil
	}

	// Get effective limit (10% for soft suspended)
	effectiveLimit := c.getEffectiveLimit(ctx, userID)

	key := "ratelimit:req:" + userID
	result, err := c.limiter.Check(ctx, key, effectiveLimit, c.window)
	if err != nil {
		// Fail open on rate limiter error
		return Allowed(), nil
	}

	if !result.Allowed {
		return Blocked(ReasonRateLimited, PriorityRateLimit).
			WithMeta("limit", strconv.Itoa(effectiveLimit)).
			WithMeta("remaining", "0").
			WithMeta("reset_at", strconv.FormatInt(result.ResetAt.Unix(), 10)), nil
	}

	return Allowed().
		WithMeta("ratelimit_remaining", strconv.Itoa(result.Remaining)).
		WithMeta("ratelimit_limit", strconv.Itoa(effectiveLimit)), nil
}

// getEffectiveLimit returns the rate limit, reduced for soft suspended users.
func (c *RateLimitChecker) getEffectiveLimit(ctx context.Context, userID string) int {
	if c.repCache == nil {
		return c.baseLimit
	}

	status, err := c.repCache.Get(ctx, userID)
	if err != nil || status == nil {
		return c.baseLimit
	}

	// Soft suspended users get 10% of normal limit
	if status.IsSoftSuspended {
		reduced := c.baseLimit / 10
		if reduced < 1 {
			reduced = 1
		}
		return reduced
	}

	return c.baseLimit
}
