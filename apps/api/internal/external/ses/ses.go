package ses

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

// IdentityResult contains the result of an email identity operation.
type IdentityResult struct {
	// Whether the identity is verified and can send emails.
	VerifiedForSendingStatus bool

	// DKIM verification status: PENDING, SUCCESS, FAILED, TEMPORARY_FAILURE, NOT_STARTED.
	DkimStatus string

	// DKIM tokens for CNAME records (3 tokens).
	DkimTokens []string

	// Custom MAIL FROM domain if configured.
	MailFromDomain string

	// MAIL FROM status: PENDING, SUCCESS, FAILED, TEMPORARY_FAILURE.
	MailFromStatus string
}

// Client defines the interface for AWS SES v2 operations.
// This interface is meant to be easily mockable for testing.
type Client interface {
	// CreateEmailIdentity creates a new email identity (domain) in SES.
	// Returns the DKIM tokens needed for DNS configuration.
	CreateEmailIdentity(ctx context.Context, domain string) (*IdentityResult, error)

	// GetEmailIdentity retrieves the current status of an email identity.
	GetEmailIdentity(ctx context.Context, domain string) (*IdentityResult, error)

	// DeleteEmailIdentity removes an email identity from SES.
	DeleteEmailIdentity(ctx context.Context, domain string) error

	// PutEmailIdentityMailFromAttributes configures custom MAIL FROM domain.
	// The mailFromDomain should be a subdomain of the identity domain.
	PutEmailIdentityMailFromAttributes(ctx context.Context, domain, mailFromDomain string) error

	// SendEmail sends a raw email (with MIME content).
	SendEmail(ctx context.Context, input *sesv2.SendEmailInput) (*sesv2.SendEmailOutput, error)
}
