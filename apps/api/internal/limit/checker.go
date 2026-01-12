package limit

import "context"

// Checker defines the interface for a limit check.
// Implementations run in parallel, so they must be goroutine-safe.
//
//go:generate mockgen -destination=mocks/mock_checker.go -package=mocks github.com/emailapi/api/internal/limit Checker
type Checker interface {
	// Check performs the limit check for the given user.
	// Must be goroutine-safe as checks run in parallel.
	Check(ctx context.Context, userID string) (*CheckResult, error)

	// Name returns a human-readable name for logging/debugging.
	Name() string
}

// Priority constants for different check types.
// Lower number = higher priority (shown first in error).
const (
	PrioritySuspension = 1 // Hard/soft suspension takes precedence
	PriorityRateLimit  = 2 // Rate limiting next
	PriorityDailyQuota = 3 // Daily quota
	PriorityMonthly    = 4 // Monthly quota last
)

// Reason constants for blocked results.
const (
	ReasonHardSuspended    = "account_suspended"
	ReasonSoftSuspended    = "account_restricted"
	ReasonRateLimited      = "rate_limit_exceeded"
	ReasonDailyExceeded    = "daily_limit_exceeded"
	ReasonMonthlyExceeded  = "monthly_limit_exceeded"
	ReasonCreditsExhausted = "credits_exhausted"
)
