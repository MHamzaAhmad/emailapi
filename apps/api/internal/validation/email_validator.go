package validation

import (
	"context"
	"strings"
	"sync"

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

// MXCache caches MX record lookup results.
type MXCache interface {
	HasMX(ctx context.Context, domain string) (*bool, error)
	SetMX(ctx context.Context, domain string, hasMX bool) error
}

// UserCache provides user lookup for sandbox validation.
type UserCache interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// EmailValidator validates email addresses for sending.
type EmailValidator struct {
	verifier           *emailverifier.Verifier
	domainChecker      DomainChecker
	suppressionChecker SuppressionChecker
	bodyValidator      *BodyValidator
	mxCache            MXCache
	reputationChecker  ReputationChecker
	sandboxDomain      string
	userCache          UserCache
}

// NewEmailValidator creates a new EmailValidator.
// SMTP checking is disabled, as noted by the user.
func NewEmailValidator(domainChecker DomainChecker, suppressionChecker SuppressionChecker, bodyValidator *BodyValidator, mxCache MXCache, reputationChecker ReputationChecker, sandboxDomain string, userCache UserCache) *EmailValidator {
	verifier := emailverifier.NewVerifier().
		EnableDomainSuggest() // Enable typo detection

	// Note: SMTP check is intentionally NOT enabled per user request
	// verifier.EnableSMTPCheck() - DISABLED

	return &EmailValidator{
		verifier:           verifier,
		domainChecker:      domainChecker,
		suppressionChecker: suppressionChecker,
		bodyValidator:      bodyValidator,
		mxCache:            mxCache,
		reputationChecker:  reputationChecker,
		sandboxDomain:      sandboxDomain,
		userCache:          userCache,
	}
}

// ValidateSendEmail validates all email addresses and body content in a send email request.
// Runs email validation, suppression check, and body URL safety check in parallel using errgroup.
// Returns nil if all validations pass, or ValidationErrors with all failures.
func (v *EmailValidator) ValidateSendEmail(ctx context.Context, userID, from string, to, cc, bcc []string, body, html string) error {
	// Fast path: Check reputation first (Redis lookup ~1ms)
	// This blocks suspended users immediately without expensive validation
	if v.reputationChecker != nil {
		if err := v.reputationChecker.CheckSendPermission(ctx, userID); err != nil {
			return err
		}
	}

	// Fast path: Sandbox domain recipient validation (Redis lookup ~1ms)
	// This ensures sandbox users can only send to their own verified email
	if err := v.validateSandboxRecipients(ctx, userID, from, to, cc, bcc); err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	var validationErrors *ValidationErrors
	var suppressedHashes []string
	var bodyError error

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

	// Goroutine 3: Body URL safety validation (runs in parallel)
	g.Go(func() error {
		if v.bodyValidator == nil {
			return nil
		}
		bodyError = v.bodyValidator.ValidateURLs(ctx, body, html)
		return bodyError
	})

	// Wait for all to complete
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

	// Add body errors if any
	if bodyError != nil {
		if ve, ok := bodyError.(*ValidationErrors); ok {
			for _, e := range ve.Errors {
				errors.Add(e)
			}
		}
	}

	if errors.HasErrors() {
		return errors
	}
	return nil
}

// validateAllAddresses performs all address validations in parallel.
func (v *EmailValidator) validateAllAddresses(ctx context.Context, userID, from string, to, cc, bcc []string) *ValidationErrors {
	errors := NewValidationErrors()
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Helper to safely add error
	addError := func(err *ValidationError) {
		mu.Lock()
		defer mu.Unlock()
		errors.Add(err)
	}

	// 1. Validate FROM email (domain ownership + verified for sending)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := v.validateSender(ctx, userID, from); err != nil {
			addError(err)
		}
	}()

	// 2. Validate TO recipients
	for i, email := range to {
		wg.Add(1)
		go func(idx int, e string) {
			defer wg.Done()
			if err := v.validateRecipient(ctx, e, formatField("to", idx)); err != nil {
				addError(err)
			}
		}(i, email)
	}

	// 3. Validate CC recipients
	for i, email := range cc {
		wg.Add(1)
		go func(idx int, e string) {
			defer wg.Done()
			if err := v.validateRecipient(ctx, e, formatField("cc", idx)); err != nil {
				addError(err)
			}
		}(i, email)
	}

	// 4. Validate BCC recipients
	for i, email := range bcc {
		wg.Add(1)
		go func(idx int, e string) {
			defer wg.Done()
			if err := v.validateRecipient(ctx, e, formatField("bcc", idx)); err != nil {
				addError(err)
			}
		}(i, email)
	}

	wg.Wait()
	return errors
}

// validateSender checks if the FROM email's domain is owned and verified.
func (v *EmailValidator) validateSender(ctx context.Context, userID, from string) *ValidationError {
	// Extract domain from email
	domainName := extractDomain(from)
	if domainName == "" {
		return InvalidSyntaxError("from", from)
	}

	// Fast path: Sandbox domain bypasses ownership check
	if v.sandboxDomain != "" && strings.EqualFold(domainName, v.sandboxDomain) {
		return nil
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

// validateSandboxRecipients ensures sandbox domain emails can only be sent to the user's own verified email.
// This is a fast-path check using Redis cache (~1ms latency).
func (v *EmailValidator) validateSandboxRecipients(ctx context.Context, userID, from string, to, cc, bcc []string) error {
	// Skip if sandbox not configured or not using sandbox domain
	if v.sandboxDomain == "" {
		return nil
	}

	domainName := extractDomain(from)
	if !strings.EqualFold(domainName, v.sandboxDomain) {
		return nil // Not using sandbox, proceed with normal validation
	}

	// Sandbox mode: validate recipients
	if v.userCache == nil {
		errs := NewValidationErrors()
		errs.Add(SandboxError("from", "sandbox validation not configured"))
		return errs
	}

	// Get user's verified email (cached in Redis for ~1ms lookup)
	user, err := v.userCache.GetByID(ctx, userID)
	if err != nil || user == nil {
		errs := NewValidationErrors()
		errs.Add(SandboxError("from", "unable to verify account for sandbox sending"))
		return errs
	}

	// Validate all recipients match user's email
	allRecipients := make([]string, 0, len(to)+len(cc)+len(bcc))
	allRecipients = append(allRecipients, to...)
	allRecipients = append(allRecipients, cc...)
	allRecipients = append(allRecipients, bcc...)

	for _, recipient := range allRecipients {
		if !strings.EqualFold(strings.TrimSpace(recipient), strings.TrimSpace(user.Email)) {
			errs := NewValidationErrors()
			errs.Add(SandboxError("to", "sandbox can only send to your verified email: "+user.Email))
			return errs
		}
	}

	return nil
}

// validateRecipient validates a recipient email address with MX cache support.
func (v *EmailValidator) validateRecipient(ctx context.Context, email, field string) *ValidationError {
	// First, do quick syntax check via the verifier
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

	domainName := result.Syntax.Domain

	// Fast-path: check MX cache first (sub-ms lookup)
	if v.mxCache != nil {
		if hasMX, _ := v.mxCache.HasMX(ctx, domainName); hasMX != nil {
			if !*hasMX {
				return NoMXRecordsError(field, email, domainName)
			}
			// Domain has MX records, validation passed
			return nil
		}
	}

	// Cache miss: use result from email-verifier (which already did MX lookup)
	hasMX := result.HasMxRecords

	// Cache the result for next time
	if v.mxCache != nil {
		_ = v.mxCache.SetMX(ctx, domainName, hasMX)
	}

	if !hasMX {
		return NoMXRecordsError(field, email, domainName)
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
