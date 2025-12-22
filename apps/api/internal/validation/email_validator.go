package validation

import (
	"context"
	"strings"

	emailverifier "github.com/AfterShip/email-verifier"
	"golang.org/x/sync/errgroup"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository/suppression"
)

// DomainChecker is an interface for checking domain ownership.
type DomainChecker interface {
	GetVerifiedDomainForSending(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error)
}

// SuppressionChecker checks if emails are suppressed.
type SuppressionChecker interface {
	CheckBatch(ctx context.Context, hashes []string) ([]string, error)
}

// EmailValidator validates email addresses for sending.
type EmailValidator struct {
	verifier           *emailverifier.Verifier
	domainChecker      DomainChecker
	suppressionChecker SuppressionChecker
}

// NewEmailValidator creates a new EmailValidator.
// SMTP checking is disabled, as noted by the user.
func NewEmailValidator(domainChecker DomainChecker, suppressionChecker SuppressionChecker) *EmailValidator {
	verifier := emailverifier.NewVerifier().
		EnableDomainSuggest() // Enable typo detection

	// Note: SMTP check is intentionally NOT enabled per user request
	// verifier.EnableSMTPCheck() - DISABLED

	return &EmailValidator{
		verifier:           verifier,
		domainChecker:      domainChecker,
		suppressionChecker: suppressionChecker,
	}
}

// ValidateSendEmail validates all email addresses in a send email request.
// Runs email validation and suppression check in parallel using errgroup.
// Returns nil if all validations pass, or ValidationErrors with all failures.
func (v *EmailValidator) ValidateSendEmail(ctx context.Context, userID, from string, to, cc, bcc []string) error {
	g, ctx := errgroup.WithContext(ctx)

	var validationErrors *ValidationErrors
	var suppressedHashes []string

	// Goroutine 1: Standard email validation (sender domain + recipient format)
	g.Go(func() error {
		validationErrors = v.validateAllAddresses(ctx, userID, from, to, cc, bcc)
		if validationErrors.HasErrors() {
			return validationErrors // Cancels context, stops other goroutines
		}
		return nil
	})

	// Goroutine 2: Suppression check (runs in parallel)
	g.Go(func() error {
		if v.suppressionChecker == nil {
			return nil
		}

		// Collect all recipient emails
		allRecipients := make([]string, 0, len(to)+len(cc)+len(bcc))
		allRecipients = append(allRecipients, to...)
		allRecipients = append(allRecipients, cc...)
		allRecipients = append(allRecipients, bcc...)

		if len(allRecipients) == 0 {
			return nil
		}

		// Hash all emails for lookup
		hashes := make([]string, len(allRecipients))
		for i, email := range allRecipients {
			hashes[i] = suppression.HashEmail(email)
		}

		// Single batch query to Redis
		var err error
		suppressedHashes, err = v.suppressionChecker.CheckBatch(ctx, hashes)
		return err
	})

	// Wait for both to complete
	if err := g.Wait(); err != nil {
		// If it's a validation error, return it directly
		if ve, ok := err.(*ValidationErrors); ok {
			return ve
		}
		return err
	}

	// Build combined error response
	errors := NewValidationErrors()

	// Add validation errors if any
	if validationErrors != nil && validationErrors.HasErrors() {
		for _, e := range validationErrors.Errors {
			errors.Add(e)
		}
	}

	// Add suppression errors if any
	if len(suppressedHashes) > 0 {
		errors.Add(SuppressionError("recipients", len(suppressedHashes)))
	}

	if errors.HasErrors() {
		return errors
	}
	return nil
}

// validateAllAddresses performs all address validations (non-concurrent helper).
func (v *EmailValidator) validateAllAddresses(ctx context.Context, userID, from string, to, cc, bcc []string) *ValidationErrors {
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

	return errors
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
