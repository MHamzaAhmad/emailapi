package ses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// sesClient implements the Client interface using AWS SDK v2.
type sesClient struct {
	client *sesv2.Client
	region string
}

// NewClient creates a new SES client.
func NewClient(ctx context.Context, region string) (Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &sesClient{
		client: sesv2.NewFromConfig(cfg),
		region: region,
	}, nil
}

// CreateEmailIdentity creates a new email identity (domain) in SES.
func (c *sesClient) CreateEmailIdentity(ctx context.Context, domain string) (*IdentityResult, error) {
	input := &sesv2.CreateEmailIdentityInput{
		EmailIdentity: aws.String(domain),
	}

	output, err := c.client.CreateEmailIdentity(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create email identity: %w", err)
	}

	result := &IdentityResult{
		VerifiedForSendingStatus: output.VerifiedForSendingStatus,
	}

	// Extract DKIM attributes
	if output.DkimAttributes != nil {
		result.DkimStatus = string(output.DkimAttributes.Status)
		result.DkimTokens = output.DkimAttributes.Tokens
	}

	return result, nil
}

// GetEmailIdentity retrieves the current status of an email identity.
func (c *sesClient) GetEmailIdentity(ctx context.Context, domain string) (*IdentityResult, error) {
	input := &sesv2.GetEmailIdentityInput{
		EmailIdentity: aws.String(domain),
	}

	output, err := c.client.GetEmailIdentity(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get email identity: %w", err)
	}

	result := &IdentityResult{
		VerifiedForSendingStatus: output.VerifiedForSendingStatus,
	}

	// Extract DKIM attributes
	if output.DkimAttributes != nil {
		result.DkimStatus = string(output.DkimAttributes.Status)
		result.DkimTokens = output.DkimAttributes.Tokens
	}

	// Extract MAIL FROM attributes
	if output.MailFromAttributes != nil {
		result.MailFromDomain = aws.ToString(output.MailFromAttributes.MailFromDomain)
		result.MailFromStatus = string(output.MailFromAttributes.MailFromDomainStatus)
	}

	return result, nil
}

// DeleteEmailIdentity removes an email identity from SES.
func (c *sesClient) DeleteEmailIdentity(ctx context.Context, domain string) error {
	input := &sesv2.DeleteEmailIdentityInput{
		EmailIdentity: aws.String(domain),
	}

	_, err := c.client.DeleteEmailIdentity(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete email identity: %w", err)
	}

	return nil
}

// PutEmailIdentityMailFromAttributes configures custom MAIL FROM domain.
func (c *sesClient) PutEmailIdentityMailFromAttributes(ctx context.Context, domain, mailFromDomain string) error {
	input := &sesv2.PutEmailIdentityMailFromAttributesInput{
		EmailIdentity:       aws.String(domain),
		MailFromDomain:      aws.String(mailFromDomain),
		BehaviorOnMxFailure: types.BehaviorOnMxFailureUseDefaultValue,
	}

	_, err := c.client.PutEmailIdentityMailFromAttributes(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to set mail from attributes: %w", err)
	}

	return nil
}
