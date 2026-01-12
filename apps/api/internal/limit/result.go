// Package limit provides a unified, extensible limit checking system.
// All limit checks run in parallel for high performance.
package limit

// Result represents the merged result from all limit checkers.
type Result struct {
	// Allowed indicates if the request should proceed.
	Allowed bool `json:"allowed"`

	// Reason contains the block reason if not allowed.
	// Empty when allowed. Examples: "rate_limited", "suspended", "daily_exceeded".
	Reason string `json:"reason,omitempty"`

	// EffectiveRateLimit is the rate limit to apply (reduced for soft suspended).
	EffectiveRateLimit int `json:"effective_rate_limit"`

	// RemainingDaily shows emails remaining today (-1 = unlimited).
	RemainingDaily int64 `json:"remaining_daily"`

	// RemainingMonthly shows emails remaining this month (-1 = unlimited).
	RemainingMonthly int64 `json:"remaining_monthly"`

	// Meta contains additional details for headers or debugging.
	Meta map[string]string `json:"meta,omitempty"`
}

// CheckResult represents the result from a single limit checker.
type CheckResult struct {
	// Allowed indicates if this check passed.
	Allowed bool

	// Reason for blocking (empty if allowed).
	Reason string

	// Priority determines which failure reason takes precedence.
	// Lower number = higher priority. Suspension (1) > Rate (2) > Quota (3).
	Priority int

	// Meta for headers or debugging.
	Meta map[string]string
}

// Blocked creates a CheckResult indicating the request is blocked.
func Blocked(reason string, priority int) *CheckResult {
	return &CheckResult{
		Allowed:  false,
		Reason:   reason,
		Priority: priority,
		Meta:     make(map[string]string),
	}
}

// Allowed creates a CheckResult indicating the check passed.
func Allowed() *CheckResult {
	return &CheckResult{
		Allowed:  true,
		Priority: 100, // Low priority, won't override failures
		Meta:     make(map[string]string),
	}
}

// WithMeta adds metadata to the result.
func (r *CheckResult) WithMeta(key, value string) *CheckResult {
	if r.Meta == nil {
		r.Meta = make(map[string]string)
	}
	r.Meta[key] = value
	return r
}
