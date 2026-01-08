package validation

import (
	"fmt"
	"strings"

	"github.com/emailapi/api/internal/domain"
)

// ValidationErrors holds multiple validation errors.
// This allows collecting all validation failures before returning.
type ValidationErrors struct {
	Errors []*domain.AppError `json:"errors"`
}

func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	var msgs []string
	for _, e := range ve.Errors {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors.
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// Add appends a validation error.
func (ve *ValidationErrors) Add(err *domain.AppError) {
	ve.Errors = append(ve.Errors, err)
}

// First returns the first error, or nil if empty.
func (ve *ValidationErrors) First() *domain.AppError {
	if len(ve.Errors) == 0 {
		return nil
	}
	return ve.Errors[0]
}

// NewValidationErrors creates a new empty ValidationErrors.
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{Errors: make([]*domain.AppError, 0)}
}

// === Error Factory Functions ===
// These create domain.AppError instances with appropriate codes.

// InvalidSyntaxError creates a syntax validation error.
func InvalidSyntaxError(field, email string) *domain.AppError {
	return domain.ErrInvalidEmailSyntax.Clone().
		WithField(field).
		WithMeta("email", email)
}

// NoMXRecordsError creates an MX records validation error.
func NoMXRecordsError(field, email, domainName string) *domain.AppError {
	return domain.ErrNoMXRecords.Clone().
		WithField(field).
		WithMeta("email", email).
		WithMeta("domain", domainName)
}

// DomainNotOwnedError creates a domain ownership error.
func DomainNotOwnedError(field, domainName string) *domain.AppError {
	return domain.ErrDomainNotOwned.Clone().
		WithField(field).
		WithMeta("domain", domainName)
}

// DomainNotVerifiedError creates a domain verification error.
func DomainNotVerifiedError(field, domainName string) *domain.AppError {
	return domain.ErrDomainNotVerified.Clone().
		WithField(field).
		WithMeta("domain", domainName)
}

// InvalidDomainWithSuggestion creates a domain error with a typo suggestion.
func InvalidDomainWithSuggestion(field, email, suggestion string) *domain.AppError {
	return domain.ErrEmailTypoDetected.Clone().
		WithField(field).
		WithMeta("email", email).
		WithMeta("suggestion", suggestion)
}

// SuppressionError creates an error for suppressed email addresses.
func SuppressionError(field string, count int) *domain.AppError {
	msg := fmt.Sprintf("%d recipient(s) suppressed", count)
	if count == 1 {
		msg = "recipient is suppressed"
	}
	err := domain.ErrEmailSuppressed.Clone().WithField(field)
	err.Message = msg
	err.Metadata = map[string]string{"count": fmt.Sprintf("%d", count)}
	return err
}

// SandboxError creates an error for sandbox domain restrictions.
func SandboxError(field, message string) *domain.AppError {
	err := domain.ErrSandboxRestriction.Clone().WithField(field)
	err.Message = message
	return err
}

// UnsafeURLError creates an error for a URL flagged as unsafe.
func UnsafeURLError(field, urlVal, threatType string) *domain.AppError {
	return domain.ErrUnsafeURL.Clone().
		WithField(field).
		WithMeta("url", urlVal).
		WithMeta("threat_type", threatType)
}
