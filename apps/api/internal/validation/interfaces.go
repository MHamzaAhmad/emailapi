package validation

//go:generate mockgen -destination=mocks/mock_validation.go -package=mocks github.com/emailapi/api/internal/validation DomainChecker,SuppressionChecker,MXCache,UserCache,BodyValidatorInterface

import (
	"context"

	"github.com/emailapi/api/internal/domain"
)

// DomainChecker is an interface for checking domain ownership.
type DomainChecker interface {
	GetVerifiedDomainForSending(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error)
}

// SuppressionChecker checks if emails are suppressed.
type SuppressionChecker interface {
	CheckBatch(ctx context.Context, hashes []string) ([]string, error)
}

// MXCache caches MX record lookup results.
type MXCache interface {
	HasMX(ctx context.Context, domain string) (*bool, error)
	SetMX(ctx context.Context, domain string, hasMX bool) error
}

// UserCache provides user lookup for sandbox validation.
type UserCache interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// BodyValidatorInterface defines body validation for mockability.
type BodyValidatorInterface interface {
	ValidateURLs(ctx context.Context, body, html string) error
}

// Ensure BodyValidator implements the interface
var _ BodyValidatorInterface = (*BodyValidator)(nil)
