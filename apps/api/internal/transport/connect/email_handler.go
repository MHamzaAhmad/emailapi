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

// EmailHandler implements the Connect EmailServiceHandler.
type EmailHandler struct {
	emailapiv1connect.UnimplementedEmailServiceHandler
	svc *service.EmailService
}

// NewEmailHandler creates a new EmailHandler.
func NewEmailHandler(svc *service.EmailService) *EmailHandler {
	return &EmailHandler{svc: svc}
}

// SendEmail handles sending an email.
func (h *EmailHandler) SendEmail(
	ctx context.Context,
	req *connect.Request[emailapiv1.SendEmailRequest],
) (*connect.Response[emailapiv1.SendEmailResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	// Add user_id to context for service layer (backwards compatibility)
	ctx = context.WithValue(ctx, "user_id", userID)

	result, err := h.svc.SendEmail(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(result), nil
}
