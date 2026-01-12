package limit

import (
	"context"
	"fmt"
)

// MonthlyQuotaChecker checks monthly email limits.
// Uses Polar credit balance as source of truth for free users.
// Paid users have usage-based billing (no hard limit).
type MonthlyQuotaChecker struct {
	creditCache CreditCacheInterface
}

// NewMonthlyQuotaChecker creates a new monthly quota checker.
func NewMonthlyQuotaChecker(creditCache CreditCacheInterface) *MonthlyQuotaChecker {
	return &MonthlyQuotaChecker{creditCache: creditCache}
}

// Name returns the checker name.
func (c *MonthlyQuotaChecker) Name() string { return "monthly_quota" }

// Check verifies user is within monthly limit.
func (c *MonthlyQuotaChecker) Check(ctx context.Context, userID string) (*CheckResult, error) {
	if c.creditCache == nil {
		return Allowed(), nil
	}

	state, err := c.creditCache.GetState(ctx, userID)
	if err != nil {
		// Fail open on cache error
		return Allowed(), nil
	}

	if state == nil {
		return Allowed(), nil
	}

	// Paid users: usage-based billing, no hard limit
	if state.IsPaid {
		return Allowed().WithMeta("monthly_limit", "-1"), nil
	}

	// Free users: check Polar credit balance (source of truth)
	if state.PolarBalance <= 0 {
		return Blocked(ReasonCreditsExhausted, PriorityMonthly).
			WithMeta("monthly_remaining", "0").
			WithMeta("plan", "free"), nil
	}

	return Allowed().
		WithMeta("monthly_remaining", fmt.Sprintf("%d", state.PolarBalance)).
		WithMeta("plan", state.PlanType), nil
}
