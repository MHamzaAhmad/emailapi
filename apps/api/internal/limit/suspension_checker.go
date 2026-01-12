package limit

import (
	"context"
)

// SuspensionChecker checks if user is hard or soft suspended.
type SuspensionChecker struct {
	repCache ReputationCacheInterface
}

// NewSuspensionChecker creates a new suspension checker.
func NewSuspensionChecker(repCache ReputationCacheInterface) *SuspensionChecker {
	return &SuspensionChecker{repCache: repCache}
}

// Name returns the checker name.
func (c *SuspensionChecker) Name() string { return "suspension" }

// Check verifies user is not hard suspended.
// Soft suspension is handled by other checkers reducing limits.
func (c *SuspensionChecker) Check(ctx context.Context, userID string) (*CheckResult, error) {
	if c.repCache == nil {
		return Allowed(), nil
	}

	status, err := c.repCache.Get(ctx, userID)
	if err != nil {
		// Fail open on cache error
		return Allowed(), nil
	}

	if status == nil {
		return Allowed(), nil
	}

	// Hard suspended = blocked completely
	if status.IsHardSuspended {
		return Blocked(ReasonHardSuspended, PrioritySuspension).
			WithMeta("suspension_type", "hard"), nil
	}

	// Soft suspended = allowed but with reduced limits
	// The reduction is handled by other checkers using IsSoftSuspended
	if status.IsSoftSuspended {
		return Allowed().WithMeta("suspension_type", "soft"), nil
	}

	return Allowed(), nil
}

// IsSoftSuspended is a helper to check soft suspension status.
func (c *SuspensionChecker) IsSoftSuspended(ctx context.Context, userID string) bool {
	if c.repCache == nil {
		return false
	}
	status, err := c.repCache.Get(ctx, userID)
	if err != nil || status == nil {
		return false
	}
	return status.IsSoftSuspended
}
