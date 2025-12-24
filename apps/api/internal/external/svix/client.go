package svix

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/emailapi/api/gen/v1"
	svixlib "github.com/svix/svix-webhooks/go"
	"github.com/svix/svix-webhooks/go/models"
	"google.golang.org/protobuf/proto"
)

// Event types for the email API.
// Each event type includes its proto message for schema generation.
var eventTypes = []struct {
	Name        string
	Description string
	Message     proto.Message // Proto message for JSON Schema generation
}{
	{"email.sent", "Email was accepted by SES", &v1.EmailSentEvent{}},
	{"email.delivered", "Email was delivered to recipient", &v1.EmailDeliveredEvent{}},
	{"email.failed", "Email sending failed", &v1.EmailFailedEvent{}},
	{"email.bounced", "Email bounced (hard bounce)", &v1.EmailBouncedEvent{}},
	{"email.complained", "Recipient marked email as spam", &v1.EmailComplainedEvent{}},
	{"email.rejected", "Email was rejected by SES", &v1.EmailRejectedEvent{}},
	{"email.delayed", "Email delivery was delayed", &v1.EmailDelayedEvent{}},
	{"email.reply_received", "Reply to a sent email was received", &v1.EmailReplyReceivedEvent{}},
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
// Schemas are generated from proto message definitions, making proto the single source of truth.
// Idempotent - creates if not exists, updates if already exists.
func (c *svixClient) EnsureEventTypes(ctx context.Context) error {
	for _, et := range eventTypes {
		desc := et.Description

		// Generate JSON schema from proto message
		schema := ProtoToJSONSchema(et.Message)

		// Schemas map uses version "1" as the default schema version
		// Type must be map[string]any for Svix SDK
		schemas := map[string]any{
			"1": schema,
		}

		_, err := c.client.EventType.Create(ctx, models.EventTypeIn{
			Name:        et.Name,
			Description: desc,
			Schemas:     &schemas,
		}, nil)
		if err != nil {
			// Ignore 409 conflict (already exists) - but try to update the schema
			if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "already exists") {
				// Update existing event type with latest schema
				_, updateErr := c.client.EventType.Update(ctx, et.Name, models.EventTypeUpdate{
					Description: desc,
					Schemas:     &schemas,
				})
				if updateErr != nil {
					return fmt.Errorf("failed to update event type %s: %w", et.Name, updateErr)
				}
				continue
			}
			return fmt.Errorf("failed to register event type %s: %w", et.Name, err)
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
