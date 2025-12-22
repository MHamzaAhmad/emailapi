package validation

import (
	"fmt"
	"strings"
)

// ValidationErrorCode represents the type of validation error.
type ValidationErrorCode string

const (
	// Syntax errors
	ErrCodeInvalidSyntax ValidationErrorCode = "INVALID_SYNTAX"
	ErrCodeInvalidDomain ValidationErrorCode = "INVALID_DOMAIN"

	// MX record errors
	ErrCodeNoMXRecords ValidationErrorCode = "NO_MX_RECORDS"

	// Domain ownership errors
	ErrCodeDomainNotOwned    ValidationErrorCode = "DOMAIN_NOT_OWNED"
	ErrCodeDomainNotVerified ValidationErrorCode = "DOMAIN_NOT_VERIFIED"

	// Suppression errors
	ErrCodeEmailSuppressed ValidationErrorCode = "EMAIL_SUPPRESSED"
)

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string              `json:"field"`
	Code    ValidationErrorCode `json:"code"`
	Message string              `json:"message"`
	// Suggestion provides a corrected value (e.g., for typo detection)
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors holds multiple validation errors.
type ValidationErrors struct {
	Errors []*ValidationError `json:"errors"`
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
func (ve *ValidationErrors) Add(err *ValidationError) {
	ve.Errors = append(ve.Errors, err)
}

// NewValidationErrors creates a new empty ValidationErrors.
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{Errors: make([]*ValidationError, 0)}
}

// InvalidSyntaxError creates a syntax validation error.
func InvalidSyntaxError(field, email string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Code:    ErrCodeInvalidSyntax,
		Message: fmt.Sprintf("'%s' is not a valid email address", email),
	}
}

// NoMXRecordsError creates an MX records validation error.
func NoMXRecordsError(field, email, domain string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Code:    ErrCodeNoMXRecords,
		Message: fmt.Sprintf("domain '%s' has no MX records - cannot receive emails", domain),
	}
}

// DomainNotOwnedError creates a domain ownership error.
func DomainNotOwnedError(field, domain string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Code:    ErrCodeDomainNotOwned,
		Message: fmt.Sprintf("domain '%s' is not registered with your account", domain),
	}
}

// DomainNotVerifiedError creates a domain verification error.
func DomainNotVerifiedError(field, domain string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Code:    ErrCodeDomainNotVerified,
		Message: fmt.Sprintf("domain '%s' is not verified for sending emails", domain),
	}
}

// InvalidDomainWithSuggestion creates a domain error with a typo suggestion.
func InvalidDomainWithSuggestion(field, email, suggestion string) *ValidationError {
	return &ValidationError{
		Field:      field,
		Code:       ErrCodeInvalidDomain,
		Message:    fmt.Sprintf("'%s' may have a typo in the domain", email),
		Suggestion: suggestion,
	}
}

// SuppressionError creates an error for suppressed email addresses.
func SuppressionError(field string, count int) *ValidationError {
	msg := fmt.Sprintf("%d recipient(s) are suppressed due to previous bounces or complaints", count)
	if count == 1 {
		msg = "recipient is suppressed due to previous bounce or complaint"
	}
	return &ValidationError{
		Field:   field,
		Code:    ErrCodeEmailSuppressed,
		Message: msg,
	}
}
