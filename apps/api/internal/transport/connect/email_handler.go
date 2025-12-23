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

// EmailHandler implements the Connect EmailServiceHandler.
type EmailHandler struct {
	v1connect.UnimplementedEmailServiceHandler
	svc *service.EmailService
}

// NewEmailHandler creates a new EmailHandler.
func NewEmailHandler(svc *service.EmailService) *EmailHandler {
	return &EmailHandler{svc: svc}
}

// SendEmail handles sending an email.
func (h *EmailHandler) SendEmail(
	ctx context.Context,
	req *connect.Request[v1.SendEmailRequest],
) (*connect.Response[v1.SendEmailResponse], error) {
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

// StreamEvents streams email events to the client.
func (h *EmailHandler) StreamEvents(
	ctx context.Context,
	req *connect.Request[v1.StreamEventsRequest],
	stream *connect.ServerStream[v1.Event],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	batchSize := req.Msg.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	if batchSize > 100 {
		batchSize = 100
	}

	events, err := h.svc.StreamEvents(ctx, userID, req.Msg.Cursor, req.Msg.EventTypes, batchSize)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil // Channel closed
			}
			if err := stream.Send(event); err != nil {
				return err
			}
		}
	}
}
