package grpc

import (
	"context"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/service"
)

// InternalServer wraps InternalService for gRPC.
type InternalServer struct {
	emailapiv1.UnimplementedInternalServiceServer
	svc             *service.InternalService
	inboundEmailSvc *service.InboundEmailService
}

// NewInternalServer creates a new InternalServer.
func NewInternalServer(svc *service.InternalService, inboundEmailSvc *service.InboundEmailService) *InternalServer {
	return &InternalServer{
		svc:             svc,
		inboundEmailSvc: inboundEmailSvc,
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

// HandleSNSNotification processes SNS notifications for inbound emails.
func (s *InternalServer) HandleSNSNotification(ctx context.Context, req *emailapiv1.SNSNotificationRequest) (*emailapiv1.SNSNotificationResponse, error) {
	if s.inboundEmailSvc == nil {
		return &emailapiv1.SNSNotificationResponse{
			Success: false,
			Message: "Inbound email service not configured",
		}, nil
	}

	err := s.inboundEmailSvc.HandleSNSNotification(ctx, req.Type, req.Message, req.SubscribeUrl)
	if err != nil {
		return &emailapiv1.SNSNotificationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &emailapiv1.SNSNotificationResponse{
		Success: true,
		Message: "SNS notification processed successfully",
	}, nil
}
