package service

import (
	"context"
	"fmt"

	"github.com/emailapi/api/internal/external/svix"
)

// WebhookService handles webhook-related operations including Svix App Portal access.
type WebhookService struct {
	svixClient svix.Client
}

// NewWebhookService creates a new WebhookService.
func NewWebhookService(svixClient svix.Client) *WebhookService {
	return &WebhookService{svixClient: svixClient}
}

// GetAppPortalAccess returns the magic URL for embedding Svix App Portal.
// The frontend uses this with svix-react to let users manage their webhook endpoints.
func (s *WebhookService) GetAppPortalAccess(ctx context.Context, userID string) (url, token string, err error) {
	// Ensure app exists for this user
	userName := "User " + userID
	if err := s.svixClient.EnsureApp(ctx, userID, userName); err != nil {
		return "", "", fmt.Errorf("failed to ensure svix app: %w", err)
	}

	// Get the app portal access URL
	url, token, err = s.svixClient.GetAppPortalAccess(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get app portal access: %w", err)
	}

	return url, token, nil
}
