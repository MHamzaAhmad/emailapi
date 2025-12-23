package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
)

// WebhookHandler implements the Connect WebhookServiceHandler.
type WebhookHandler struct {
	v1connect.UnimplementedWebhookServiceHandler
	svc *service.WebhookService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(svc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// GetAppPortalAccess returns a magic URL for embedding Svix App Portal.
func (h *WebhookHandler) GetAppPortalAccess(
	ctx context.Context,
	req *connect.Request[v1.GetAppPortalAccessRequest],
) (*connect.Response[v1.GetAppPortalAccessResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	url, token, err := h.svc.GetAppPortalAccess(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.GetAppPortalAccessResponse{
		Url:   url,
		Token: token,
	}), nil
}
