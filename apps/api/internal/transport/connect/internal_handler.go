package connect

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/service"
	"github.com/emailapi/api/internal/transport/connect/interceptor"
	"github.com/rs/zerolog/log"
)

// InternalHandler implements the Connect InternalServiceHandler.
type InternalHandler struct {
	v1connect.UnimplementedInternalServiceHandler
	svc *service.InternalService
}

// NewInternalHandler creates a new InternalHandler.
func NewInternalHandler(svc *service.InternalService) *InternalHandler {
	return &InternalHandler{svc: svc}
}

// HandleGuardDutyScanResult processes GuardDuty malware scan results from EventBridge.
func (h *InternalHandler) HandleGuardDutyScanResult(
	ctx context.Context,
	req *connect.Request[v1.GuardDutyScanResultRequest],
) (*connect.Response[v1.GuardDutyScanResultResponse], error) {
	result := &service.GuardDutyScanResult{
		S3Bucket:   req.Msg.S3Bucket,
		S3Key:      req.Msg.S3Key,
		ScanStatus: req.Msg.ScanStatus,
		ThreatName: req.Msg.ThreatName,
	}

	err := h.svc.HandleGuardDutyScanResult(ctx, result)
	if err != nil {
		return connect.NewResponse(&v1.GuardDutyScanResultResponse{
			Success: false,
			Message: err.Error(),
		}), nil
	}

	return connect.NewResponse(&v1.GuardDutyScanResultResponse{
		Success: true,
		Message: "Scan result processed successfully",
	}), nil
}

// HandleClerkWebhook processes Clerk user lifecycle events.
func (h *InternalHandler) HandleClerkWebhook(
	ctx context.Context,
	req *connect.Request[v1.ClerkWebhookRequest],
) (*connect.Response[v1.ClerkWebhookResponse], error) {
	// Import the interceptor package to access GetRawBody
	// Get the raw body from context (captured by middleware before Connect unmarshaling)
	rawBody := interceptor.GetRawBody(ctx)
	if rawBody == nil {
		log.Error().Msg("HandleClerkWebhook: No raw body in context - middleware not working?")
		return connect.NewResponse(&v1.ClerkWebhookResponse{
			Success: false,
			Message: "Internal error: raw body not captured",
		}), nil
	}

	// Collect headers
	headers := http.Header{}
	for k, values := range req.Header() {
		for _, v := range values {
			headers.Add(k, v)
		}
	}

	// Call service with RAW body (not the unmarshaled proto payload)
	success, message, err := h.svc.HandleClerkWebhook(ctx, rawBody, headers)
	if err != nil {
		return connect.NewResponse(&v1.ClerkWebhookResponse{
			Success: false,
			Message: err.Error(),
		}), nil
	}

	return connect.NewResponse(&v1.ClerkWebhookResponse{
		Success: success,
		Message: message,
	}), nil
}

// HandlePolarWebhook processes Polar subscription webhooks.
func (h *InternalHandler) HandlePolarWebhook(
	ctx context.Context,
	req *connect.Request[v1.PolarWebhookRequest],
) (*connect.Response[v1.PolarWebhookResponse], error) {
	// Import the interceptor package to access GetRawBody
	// Get the raw body from context (captured by middleware before Connect unmarshaling)
	rawBody := interceptor.GetRawBody(ctx)
	if rawBody == nil {
		log.Error().Msg("HandlePolarWebhook: No raw body in context - middleware not working?")
		return connect.NewResponse(&v1.PolarWebhookResponse{
			Success: false,
			Message: "Internal error: raw body not captured",
		}), nil
	}

	// Collect headers
	headers := http.Header{}
	for k, values := range req.Header() {
		for _, v := range values {
			headers.Add(k, v)
		}
	}

	// Call service with RAW body
	success, message, err := h.svc.HandlePolarWebhook(ctx, rawBody, headers)
	if err != nil {
		return connect.NewResponse(&v1.PolarWebhookResponse{
			Success: false,
			Message: err.Error(),
		}), nil
	}

	return connect.NewResponse(&v1.PolarWebhookResponse{
		Success: success,
		Message: message,
	}), nil
}
