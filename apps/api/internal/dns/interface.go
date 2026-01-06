package dns

import "context"

//go:generate mockgen -destination=mocks/mock_dns.go -package=mocks github.com/emailapi/api/internal/dns ValidatorInterface

// ValidatorInterface defines the contract for DNS validation.
type ValidatorInterface interface {
	ValidateRecords(ctx context.Context, expected []ExpectedRecord) *ValidationResult
}
