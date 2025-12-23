package grpc

import (
	"context"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/service"
)

// InternalServer wraps InternalService for gRPC.
type InternalServer struct {
	emailapiv1.UnimplementedInternalServiceServer
	svc *service.InternalService
}

// NewInternalServer creates a new InternalServer.
func NewInternalServer(svc *service.InternalService) *InternalServer {
	return &InternalServer{
		svc: svc,
	}
}

// HandleGuardDutyScanResult processes GuardDuty malware scan results from EventBridge.
func (s *InternalServer) HandleGuardDutyScanResult(ctx context.Context, req *emailapiv1.GuardDutyScanResultRequest) (*emailapiv1.GuardDutyScanResultResponse, error) {
	result := &service.GuardDutyScanResult{
		S3Bucket:   req.S3Bucket,
		S3Key:      req.S3Key,
		ScanStatus: req.ScanStatus,
		ThreatName: req.ThreatName,
	}

	err := s.svc.HandleGuardDutyScanResult(ctx, result)
	if err != nil {
		return &emailapiv1.GuardDutyScanResultResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &emailapiv1.GuardDutyScanResultResponse{
		Success: true,
		Message: "Scan result processed successfully",
	}, nil
}

// HandleClerkWebhook processes Clerk user lifecycle events.
func (s *InternalServer) HandleClerkWebhook(ctx context.Context, req *emailapiv1.ClerkWebhookRequest) (*emailapiv1.ClerkWebhookResponse, error) {
	result := &service.ClerkWebhookRequest{
		Payload:       req.Payload,
		SvixID:        req.SvixId,
		SvixTimestamp: req.SvixTimestamp,
		SvixSignature: req.SvixSignature,
	}

	success, message, err := s.svc.HandleClerkWebhook(ctx, result)
	if err != nil {
		return &emailapiv1.ClerkWebhookResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &emailapiv1.ClerkWebhookResponse{
		Success: success,
		Message: message,
	}, nil
}
