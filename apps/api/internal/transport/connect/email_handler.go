package connect

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
	transporterrors "github.com/emailapi/api/internal/transport/errors"
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
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	// Add user_id to context for service layer (backwards compatibility)
	ctx = context.WithValue(ctx, "user_id", userID)

	result, err := h.svc.SendEmail(ctx, req.Msg)
	if err != nil {
		return nil, transporterrors.ToConnectError(err)
	}

	return connect.NewResponse(result), nil
}

// StreamEvents streams email events to the client using Redis Consumer Groups.
// Uses the API key ID as the consumer identifier for at-least-once delivery.
// Unacknowledged events are automatically replayed on reconnect.
// Sends heartbeat events every 25 seconds to keep the connection alive.
func (h *EmailHandler) StreamEvents(
	ctx context.Context,
	req *connect.Request[v1.StreamEventsRequest],
	stream *connect.ServerStream[v1.Event],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	apiKeyID := interceptor.GetAPIKeyID(ctx)
	if apiKeyID == "" {
		return transporterrors.ToConnectError(domain.ErrInvalidAPIKey)
	}

	batchSize := req.Msg.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	if batchSize > 100 {
		batchSize = 100
	}

	events, err := h.svc.StreamEvents(ctx, userID, apiKeyID, req.Msg.EventTypes, batchSize)
	if err != nil {
		return transporterrors.ToConnectError(err)
	}

	// Heartbeat ticker - sends keepalive every 25 seconds (before 31s WriteTimeout)
	heartbeatTicker := time.NewTicker(25 * time.Second)
	defer heartbeatTicker.Stop()

	heartbeatCounter := int64(0)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-heartbeatTicker.C:
			// Send heartbeat event to keep connection alive
			heartbeatCounter++
			heartbeat := &v1.Event{
				Id:        "heartbeat-" + string(rune(heartbeatCounter)),
				Type:      v1.EventType_EVENT_TYPE_HEARTBEAT,
				Timestamp: timestamppb.Now(),
			}
			if err := stream.Send(heartbeat); err != nil {
				return err
			}

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

// AckEvents acknowledges events as processed, preventing replay on reconnect.
func (h *EmailHandler) AckEvents(
	ctx context.Context,
	req *connect.Request[v1.AckEventsRequest],
) (*connect.Response[v1.AckEventsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, transporterrors.ToConnectError(domain.ErrUnauthenticated)
	}

	if len(req.Msg.EventIds) == 0 {
		return connect.NewResponse(&v1.AckEventsResponse{AckedCount: 0}), nil
	}

	acked, err := h.svc.AckEvents(ctx, userID, req.Msg.EventIds)
	if err != nil {
		return nil, transporterrors.ToConnectError(err)
	}

	return connect.NewResponse(&v1.AckEventsResponse{
		AckedCount: int32(acked),
	}), nil
}
