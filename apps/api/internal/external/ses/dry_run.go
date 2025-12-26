package ses

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/google/uuid"
)

// dryRunClient wraps a real SES client and intercepts SendEmail calls
// when either:
// 1. The context contains dry_run=true (for sync requests with X-Dry-Run header)
// 2. The DRY_RUN environment variable is set to "true" (for async queue workers)
//
// Used for performance testing without actually sending emails.
type dryRunClient struct {
	wrapped      Client
	globalDryRun bool
}

// NewDryRunClient wraps an existing SES client with dry-run support.
// When dry-run mode is active, SendEmail will:
// - Simulate realistic SES latency (50-150ms)
// - Return a fake message ID
// - NOT actually send the email
//
// Dry-run is triggered by:
// - Context containing "dry_run" = true (per-request)
// - DRY_RUN=true environment variable (global)
//
// All other operations (identity management) pass through to the real client.
func NewDryRunClient(client Client) Client {
	globalDryRun := os.Getenv("DRY_RUN") == "true"
	return &dryRunClient{
		wrapped:      client,
		globalDryRun: globalDryRun,
	}
}

// CreateEmailIdentity passes through to the wrapped client.
func (c *dryRunClient) CreateEmailIdentity(ctx context.Context, domain string) (*IdentityResult, error) {
	return c.wrapped.CreateEmailIdentity(ctx, domain)
}

// GetEmailIdentity passes through to the wrapped client.
func (c *dryRunClient) GetEmailIdentity(ctx context.Context, domain string) (*IdentityResult, error) {
	return c.wrapped.GetEmailIdentity(ctx, domain)
}

// DeleteEmailIdentity passes through to the wrapped client.
func (c *dryRunClient) DeleteEmailIdentity(ctx context.Context, domain string) error {
	return c.wrapped.DeleteEmailIdentity(ctx, domain)
}

// PutEmailIdentityMailFromAttributes passes through to the wrapped client.
func (c *dryRunClient) PutEmailIdentityMailFromAttributes(ctx context.Context, domain, mailFromDomain string) error {
	return c.wrapped.PutEmailIdentityMailFromAttributes(ctx, domain, mailFromDomain)
}

// PutEmailIdentityConfigurationSetAttributes passes through to the wrapped client.
func (c *dryRunClient) PutEmailIdentityConfigurationSetAttributes(ctx context.Context, domain, configurationSetName string) error {
	return c.wrapped.PutEmailIdentityConfigurationSetAttributes(ctx, domain, configurationSetName)
}

// SendEmail intercepts the call and returns a fake response if dry-run mode is enabled.
// Otherwise, passes through to the real SES client.
func (c *dryRunClient) SendEmail(ctx context.Context, input *sesv2.SendEmailInput) (*sesv2.SendEmailOutput, error) {
	// Check for dry-run mode:
	// 1. Per-request: context contains dry_run=true
	// 2. Global: DRY_RUN environment variable
	isDryRun := c.globalDryRun
	if !isDryRun {
		if ctxDryRun, ok := ctx.Value("dry_run").(bool); ok && ctxDryRun {
			isDryRun = true
		}
	}

	if isDryRun {
		// Simulate realistic SES latency (50-150ms random delay)
		delay := 50*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
		time.Sleep(delay)

		// Generate a fake message ID that looks realistic
		fakeMessageID := fmt.Sprintf("dry-run-%s@simpleemailapi.dev", uuid.New().String()[:8])

		return &sesv2.SendEmailOutput{
			MessageId: &fakeMessageID,
		}, nil
	}

	// Not dry-run, pass through to real client
	return c.wrapped.SendEmail(ctx, input)
}
