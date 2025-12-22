package grpc

import (
	"context"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/middleware"
	"github.com/emailapi/api/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WebhookServer wraps WebhookService for gRPC.
type WebhookServer struct {
	emailapiv1.UnimplementedWebhookServiceServer
	svc *service.WebhookService
}

// NewWebhookServer creates a new WebhookServer.
func NewWebhookServer(svc *service.WebhookService) *WebhookServer {
	return &WebhookServer{svc: svc}
}

// GetAppPortalAccess returns a magic URL for embedding Svix App Portal.
func (s *WebhookServer) GetAppPortalAccess(ctx context.Context, req *emailapiv1.GetAppPortalAccessRequest) (*emailapiv1.GetAppPortalAccessResponse, error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	url, token, err := s.svc.GetAppPortalAccess(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get app portal access: %v", err)
	}

	return &emailapiv1.GetAppPortalAccessResponse{
		Url:   url,
		Token: token,
	}, nil
}

// CreateWebhook is a no-op since webhook endpoints are managed via Svix App Portal.
func (s *WebhookServer) CreateWebhook(ctx context.Context, req *emailapiv1.CreateWebhookRequest) (*emailapiv1.CreateWebhookResponse, error) {
	return nil, status.Error(codes.Unimplemented, "webhook endpoints are managed via the App Portal")
}

// GetWebhook is a no-op since webhook endpoints are managed via Svix App Portal.
func (s *WebhookServer) GetWebhook(ctx context.Context, req *emailapiv1.GetWebhookRequest) (*emailapiv1.Webhook, error) {
	return nil, status.Error(codes.Unimplemented, "webhook endpoints are managed via the App Portal")
}

// ListWebhooks is a no-op since webhook endpoints are managed via Svix App Portal.
func (s *WebhookServer) ListWebhooks(ctx context.Context, req *emailapiv1.ListWebhooksRequest) (*emailapiv1.ListWebhooksResponse, error) {
	return nil, status.Error(codes.Unimplemented, "webhook endpoints are managed via the App Portal")
}

// UpdateWebhook is a no-op since webhook endpoints are managed via Svix App Portal.
func (s *WebhookServer) UpdateWebhook(ctx context.Context, req *emailapiv1.UpdateWebhookRequest) (*emailapiv1.Webhook, error) {
	return nil, status.Error(codes.Unimplemented, "webhook endpoints are managed via the App Portal")
}

// DeleteWebhook is a no-op since webhook endpoints are managed via Svix App Portal.
func (s *WebhookServer) DeleteWebhook(ctx context.Context, req *emailapiv1.DeleteWebhookRequest) (*emailapiv1.DeleteWebhookResponse, error) {
	return nil, status.Error(codes.Unimplemented, "webhook endpoints are managed via the App Portal")
}
