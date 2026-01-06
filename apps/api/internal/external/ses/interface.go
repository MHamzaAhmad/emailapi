package ses

//go:generate mockgen -destination=mocks/mock_ses.go -package=mocks github.com/emailapi/api/internal/external/ses Client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

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

	// PutEmailIdentityConfigurationSetAttributes applies a configuration set to an identity.
	// Used to enable delivery, bounce, and complaint notifications via SNS.
	PutEmailIdentityConfigurationSetAttributes(ctx context.Context, domain, configurationSetName string) error

	// SendEmail sends a raw email (with MIME content).
	SendEmail(ctx context.Context, input *sesv2.SendEmailInput) (*sesv2.SendEmailOutput, error)
}
