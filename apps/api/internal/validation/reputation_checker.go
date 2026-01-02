package validation

import "context"

//go:generate mockgen -destination=mocks/mock_reputation_checker.go -package=mocks github.com/emailapi/api/internal/validation ReputationChecker

// ReputationChecker checks user reputation for email sending permission.
type ReputationChecker interface {
	// CheckSendPermission checks if a user is allowed to send emails.
	// Returns an error if the user is suspended.
	CheckSendPermission(ctx context.Context, userID string) error

	// GetEffectiveRateLimit returns the effective rate limit for a user.
	// Flagged users get reduced limits (10% of normal).
	GetEffectiveRateLimit(ctx context.Context, userID string, baseLimit int) int
}
