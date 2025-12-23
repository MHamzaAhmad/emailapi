package connect

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/service"
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
	// Connect provides headers directly from the request
	headers := http.Header{}
	for k, values := range req.Header() {
		for _, v := range values {
			headers.Add(k, v)
		}
	}

	success, message, err := h.svc.HandleClerkWebhook(ctx, req.Msg.Payload, headers)
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
