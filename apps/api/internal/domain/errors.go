package domain

import (
	"errors"
	"fmt"

	v1 "github.com/emailapi/api/gen/v1"
)

// AppError is the standard application error type.
// It carries a typed error code for client-side matching.
type AppError struct {
	Code     v1.ErrorCode      // Typed error code (proto-generated)
	Message  string            // Human-readable message
	Field    string            // For validation errors
	Metadata map[string]string // Additional context
	Cause    error             // Underlying cause (not exposed to clients)
}

func (e *AppError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewError creates a new AppError.
func NewError(code v1.ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// WithField adds field context (for validation errors).
func (e *AppError) WithField(field string) *AppError {
	e.Field = field
	return e
}

// WithMeta adds metadata context.
func (e *AppError) WithMeta(key, value string) *AppError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// WithCause wraps an underlying error.
func (e *AppError) WithCause(err error) *AppError {
	e.Cause = err
	return e
}

// Clone creates a copy of the error (for thread-safety when adding context).
func (e *AppError) Clone() *AppError {
	clone := &AppError{
		Code:    e.Code,
		Message: e.Message,
		Field:   e.Field,
		Cause:   e.Cause,
	}
	if e.Metadata != nil {
		clone.Metadata = make(map[string]string, len(e.Metadata))
		for k, v := range e.Metadata {
			clone.Metadata[k] = v
		}
	}
	return clone
}

// === Predefined Errors ===
// Use Clone() when adding context to avoid mutating the original.

var (
	// Authentication (1xx)
	ErrUnauthenticated   = NewError(v1.ErrorCode_ERROR_CODE_UNAUTHENTICATED, "authentication required")
	ErrInvalidAPIKey     = NewError(v1.ErrorCode_ERROR_CODE_INVALID_API_KEY, "invalid API key")
	ErrExpiredAPIKey     = NewError(v1.ErrorCode_ERROR_CODE_EXPIRED_API_KEY, "API key has expired")
	ErrInsufficientScope = NewError(v1.ErrorCode_ERROR_CODE_INSUFFICIENT_SCOPE, "missing required scope")
	ErrClerkTokenInvalid = NewError(v1.ErrorCode_ERROR_CODE_CLERK_TOKEN_INVALID, "invalid Clerk token")

	// Authorization (2xx)
	ErrPermissionDenied = NewError(v1.ErrorCode_ERROR_CODE_PERMISSION_DENIED, "permission denied")
	ErrAdminRequired    = NewError(v1.ErrorCode_ERROR_CODE_ADMIN_REQUIRED, "admin access required")
	ErrAccountSuspended = NewError(v1.ErrorCode_ERROR_CODE_ACCOUNT_SUSPENDED, "account suspended: contact support")

	// Validation (3xx)
	ErrInvalidArgument      = NewError(v1.ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "invalid argument")
	ErrMissingRequiredField = NewError(v1.ErrorCode_ERROR_CODE_MISSING_REQUIRED_FIELD, "required field missing")
	ErrInvalidEmailSyntax   = NewError(v1.ErrorCode_ERROR_CODE_INVALID_EMAIL_SYNTAX, "invalid email syntax")
	ErrNoMXRecords          = NewError(v1.ErrorCode_ERROR_CODE_NO_MX_RECORDS, "domain has no MX records")
	ErrEmailTypoDetected    = NewError(v1.ErrorCode_ERROR_CODE_EMAIL_TYPO_DETECTED, "possible typo in email")
	ErrUnsafeURL            = NewError(v1.ErrorCode_ERROR_CODE_UNSAFE_URL, "URL flagged as unsafe")
	ErrEmailSuppressed      = NewError(v1.ErrorCode_ERROR_CODE_EMAIL_SUPPRESSED, "email is suppressed")
	ErrSandboxRestriction   = NewError(v1.ErrorCode_ERROR_CODE_SANDBOX_RESTRICTION, "sandbox restriction")

	// Resources (4xx)
	ErrNotFound            = NewError(v1.ErrorCode_ERROR_CODE_NOT_FOUND, "resource not found")
	ErrDomainNotFound      = NewError(v1.ErrorCode_ERROR_CODE_DOMAIN_NOT_FOUND, "domain not found")
	ErrAPIKeyNotFound      = NewError(v1.ErrorCode_ERROR_CODE_API_KEY_NOT_FOUND, "API key not found")
	ErrUserNotFound        = NewError(v1.ErrorCode_ERROR_CODE_USER_NOT_FOUND, "user not found")
	ErrAlreadyExists       = NewError(v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS, "resource already exists")
	ErrDomainAlreadyExists = NewError(v1.ErrorCode_ERROR_CODE_DOMAIN_ALREADY_EXISTS, "domain already exists")

	// Domain Verification (5xx)
	ErrDomainNotOwned       = NewError(v1.ErrorCode_ERROR_CODE_DOMAIN_NOT_OWNED, "domain not registered")
	ErrDomainNotVerified    = NewError(v1.ErrorCode_ERROR_CODE_DOMAIN_NOT_VERIFIED, "domain not verified")
	ErrDomainDNSMismatch    = NewError(v1.ErrorCode_ERROR_CODE_DOMAIN_DNS_MISMATCH, "DNS record mismatch")
	ErrDomainVerifyCooldown = NewError(v1.ErrorCode_ERROR_CODE_DOMAIN_VERIFY_COOLDOWN, "verification rate limited")

	// Rate Limiting (6xx)
	ErrRateLimited          = NewError(v1.ErrorCode_ERROR_CODE_RATE_LIMITED, "rate limit exceeded")
	ErrDailyLimitExceeded   = NewError(v1.ErrorCode_ERROR_CODE_DAILY_LIMIT_EXCEEDED, "daily limit exceeded")
	ErrCreditsExhausted     = NewError(v1.ErrorCode_ERROR_CODE_MONTHLY_CREDITS_EXHAUSTED, "monthly credits exhausted")
	ErrMaxConcurrentStreams = NewError(v1.ErrorCode_ERROR_CODE_MAX_CONCURRENT_STREAMS, "too many concurrent streams")

	// Attachments (7xx)
	ErrAttachmentTooLarge     = NewError(v1.ErrorCode_ERROR_CODE_ATTACHMENT_TOO_LARGE, "attachment too large")
	ErrTotalSizeExceeded      = NewError(v1.ErrorCode_ERROR_CODE_TOTAL_SIZE_EXCEEDED, "total attachments too large")
	ErrAttachmentThreatFound  = NewError(v1.ErrorCode_ERROR_CODE_ATTACHMENT_THREAT_FOUND, "threat detected")
	ErrUnsupportedContentType = NewError(v1.ErrorCode_ERROR_CODE_UNSUPPORTED_CONTENT_TYPE, "unsupported file type")

	// Tokens (8xx)
	ErrInvalidToken         = NewError(v1.ErrorCode_ERROR_CODE_INVALID_TOKEN, "invalid token")
	ErrExpiredToken         = NewError(v1.ErrorCode_ERROR_CODE_EXPIRED_TOKEN, "token expired")
	ErrWebhookSecretInvalid = NewError(v1.ErrorCode_ERROR_CODE_WEBHOOK_SECRET_INVALID, "invalid webhook secret")

	// Internal (9xx)
	ErrInternal           = NewError(v1.ErrorCode_ERROR_CODE_INTERNAL, "internal error")
	ErrUpstreamProvider   = NewError(v1.ErrorCode_ERROR_CODE_UPSTREAM_PROVIDER_ERROR, "provider error")
	ErrServiceUnavailable = NewError(v1.ErrorCode_ERROR_CODE_SERVICE_UNAVAILABLE, "service unavailable")
)

// IsAppError checks if an error is an AppError and returns it.
func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// WrapError creates an AppError from any error with the given code.
func WrapError(code v1.ErrorCode, err error) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: err.Error(),
		Cause:   err,
	}
}
