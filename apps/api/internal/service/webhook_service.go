package service

import (
	"context"

	"github.com/emailapi/api/internal/domain"
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
		return "", "", domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "ensure_svix_app")
	}

	// Get the app portal access URL
	url, token, err = s.svixClient.GetAppPortalAccess(ctx, userID)
	if err != nil {
		return "", "", domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "get_app_portal_access")
	}

	return url, token, nil
}
