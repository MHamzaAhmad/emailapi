package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/emailapiv1connect"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
)

// WebhookHandler implements the Connect WebhookServiceHandler.
type WebhookHandler struct {
	emailapiv1connect.UnimplementedWebhookServiceHandler
	svc *service.WebhookService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(svc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// GetAppPortalAccess returns a magic URL for embedding Svix App Portal.
func (h *WebhookHandler) GetAppPortalAccess(
	ctx context.Context,
	req *connect.Request[emailapiv1.GetAppPortalAccessRequest],
) (*connect.Response[emailapiv1.GetAppPortalAccessResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	url, token, err := h.svc.GetAppPortalAccess(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emailapiv1.GetAppPortalAccessResponse{
		Url:   url,
		Token: token,
	}), nil
}
