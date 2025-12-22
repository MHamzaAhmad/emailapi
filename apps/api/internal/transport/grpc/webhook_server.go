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
// Users can use this URL with svix-react to manage their webhook endpoints.
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
