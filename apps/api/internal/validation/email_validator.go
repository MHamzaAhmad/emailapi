package validation

import (
	"context"
	"strings"

	emailverifier "github.com/AfterShip/email-verifier"

	"github.com/emailapi/api/internal/domain"
)

// DomainChecker is an interface for checking domain ownership.
type DomainChecker interface {
	GetVerifiedDomainForSending(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error)
}

// EmailValidator validates email addresses for sending.
type EmailValidator struct {
	verifier      *emailverifier.Verifier
	domainChecker DomainChecker
}

// NewEmailValidator creates a new EmailValidator.
// SMTP checking is disabled, as noted by the user.
func NewEmailValidator(domainChecker DomainChecker) *EmailValidator {
	verifier := emailverifier.NewVerifier().
		EnableDomainSuggest() // Enable typo detection

	// Note: SMTP check is intentionally NOT enabled per user request
	// verifier.EnableSMTPCheck() - DISABLED

	return &EmailValidator{
		verifier:      verifier,
		domainChecker: domainChecker,
	}
}

// ValidateSendEmail validates all email addresses in a send email request.
// Returns nil if all validations pass, or ValidationErrors with all failures.
func (v *EmailValidator) ValidateSendEmail(ctx context.Context, userID, from string, to, cc, bcc []string) error {
	errors := NewValidationErrors()

	// 1. Validate FROM email (domain ownership + verified for sending)
	if err := v.validateSender(ctx, userID, from); err != nil {
		errors.Add(err)
	}

	// 2. Validate TO recipients
	for i, email := range to {
		if err := v.validateRecipient(email, formatField("to", i)); err != nil {
			errors.Add(err)
		}
	}

	// 3. Validate CC recipients
	for i, email := range cc {
		if err := v.validateRecipient(email, formatField("cc", i)); err != nil {
			errors.Add(err)
		}
	}

	// 4. Validate BCC recipients
	for i, email := range bcc {
		if err := v.validateRecipient(email, formatField("bcc", i)); err != nil {
			errors.Add(err)
		}
	}

	if errors.HasErrors() {
		return errors
	}
	return nil
}

// validateSender checks if the FROM email's domain is owned and verified.
func (v *EmailValidator) validateSender(ctx context.Context, userID, from string) *ValidationError {
	// Extract domain from email
	domainName := extractDomain(from)
	if domainName == "" {
		return InvalidSyntaxError("from", from)
	}

	// Check if user owns this domain
	d, err := v.domainChecker.GetVerifiedDomainForSending(ctx, userID, domainName)
	if err != nil {
		return DomainNotOwnedError("from", domainName)
	}

	// Check if domain is verified for sending
	if !d.VerifiedForSending {
		return DomainNotVerifiedError("from", domainName)
	}

	return nil
}

// validateRecipient validates a recipient email address.
func (v *EmailValidator) validateRecipient(email, field string) *ValidationError {
	result, err := v.verifier.Verify(email)
	if err != nil {
		// If verification fails entirely, treat as invalid
		return InvalidSyntaxError(field, email)
	}

	// Check syntax
	if !result.Syntax.Valid {
		return InvalidSyntaxError(field, email)
	}

	// Check for domain typo suggestion (return as warning/error with suggestion)
	if result.Suggestion != "" {
		// Construct the suggested email
		suggestedEmail := result.Syntax.Username + "@" + result.Suggestion
		return InvalidDomainWithSuggestion(field, email, suggestedEmail)
	}

	// Check MX records
	if !result.HasMxRecords {
		return NoMXRecordsError(field, email, result.Syntax.Domain)
	}

	// Note: Disposable email check is intentionally skipped per user request
	// Note: Role account check is informational only, not blocking
	// Note: Free provider check is informational only, not blocking

	return nil
}

// extractDomain extracts the domain part from an email address.
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// formatField creates a field name with index for arrays.
func formatField(base string, index int) string {
	return strings.ToLower(base) + "[" + string(rune('0'+index)) + "]"
}
