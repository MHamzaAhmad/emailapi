package svix

import (
	"context"
	"fmt"
	"strings"

	svixlib "github.com/svix/svix-webhooks/go"
	"github.com/svix/svix-webhooks/go/models"
)

// Event types for the email API.
var eventTypes = []struct {
	Name        string
	Description string
}{
	{"email.sent", "Email was accepted by SES"},
	{"email.delivered", "Email was delivered to recipient"},
	{"email.failed", "Email sending failed"},
	{"email.bounced", "Email bounced (hard bounce)"},
	{"email.complained", "Recipient marked email as spam"},
	{"email.rejected", "Email was rejected by SES"},
	{"email.delayed", "Email delivery was delayed"},
	{"email.reply_received", "Reply to a sent email was received"},
}

// svixClient implements the Client interface using Svix SDK.
type svixClient struct {
	client *svixlib.Svix
}

// NewClient creates a new Svix client.
func NewClient(apiKey string) (Client, error) {
	client, err := svixlib.New(apiKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create svix client: %w", err)
	}
	return &svixClient{client: client}, nil
}

// EnsureEventTypes registers all email event types in Svix.
// Idempotent - creates if not exists, skips if already exists.
func (c *svixClient) EnsureEventTypes(ctx context.Context) error {
	for _, et := range eventTypes {
		desc := et.Description
		_, err := c.client.EventType.Create(ctx, models.EventTypeIn{
			Name:        et.Name,
			Description: desc,
		}, nil)
		if err != nil {
			// Ignore 409 conflict (already exists)
			if !strings.Contains(err.Error(), "409") && !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to register event type %s: %w", et.Name, err)
			}
		}
	}
	return nil
}

// EnsureApp creates a Svix app for the user if it doesn't exist.
func (c *svixClient) EnsureApp(ctx context.Context, userID, userName string) error {
	_, err := c.client.Application.GetOrCreate(ctx, models.ApplicationIn{
		Uid:  &userID,
		Name: userName,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to ensure svix app: %w", err)
	}
	return nil
}

// GetAppPortalAccess returns the magic URL for embedding App Portal.
func (c *svixClient) GetAppPortalAccess(ctx context.Context, userID string) (string, string, error) {
	out, err := c.client.Authentication.AppPortalAccess(ctx, userID, models.AppPortalAccessIn{}, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to get app portal access: %w", err)
	}
	return out.Url, out.Token, nil
}

// SendMessage sends a webhook message to all user's configured endpoints.
func (c *svixClient) SendMessage(ctx context.Context, userID, eventType string, payload interface{}) error {
	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		payloadMap = map[string]interface{}{"data": payload}
	}

	_, err := c.client.Message.Create(ctx, userID, models.MessageIn{
		EventType: eventType,
		Payload:   payloadMap,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to send webhook message: %w", err)
	}
	return nil
}
