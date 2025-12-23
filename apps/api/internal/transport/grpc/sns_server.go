package grpc

import (
	"context"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/service"
)

// SnsServer wraps SNSNotificationService and InboundEmailService for gRPC.
// These endpoints are public but authenticated via SNS signature verification.
type SnsServer struct {
	emailapiv1.UnimplementedSnsServiceServer
	svc        *service.SNSNotificationService
	inboundSvc *service.InboundEmailService
}

// NewSnsServer creates a new SnsServer.
func NewSnsServer(svc *service.SNSNotificationService, inboundSvc *service.InboundEmailService) *SnsServer {
	return &SnsServer{
		svc:        svc,
		inboundSvc: inboundSvc,
	}
}

// HandleSNSNotification processes SNS notifications for outbound email events.
func (s *SnsServer) HandleSNSNotification(ctx context.Context, req *emailapiv1.SNSNotificationRequest) (*emailapiv1.SNSNotificationResponse, error) {
	input := &service.SNSInput{
		Type:             req.Type,
		MessageID:        req.MessageId,
		TopicArn:         req.TopicArn,
		Message:          req.Message,
		SubscribeURL:     req.SubscribeUrl,
		Timestamp:        req.Timestamp,
		SignatureVersion: req.SignatureVersion,
		Signature:        req.Signature,
		SigningCertURL:   req.SigningCertUrl,
		Subject:          req.Subject,
		Token:            req.Token,
	}

	if err := s.svc.HandleNotification(ctx, input); err != nil {
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

// HandleInboundSNSNotification processes SNS notifications for inbound emails (replies).
func (s *SnsServer) HandleInboundSNSNotification(ctx context.Context, req *emailapiv1.SNSNotificationRequest) (*emailapiv1.SNSNotificationResponse, error) {
	if s.inboundSvc == nil {
		return &emailapiv1.SNSNotificationResponse{
			Success: false,
			Message: "Inbound email service not configured",
		}, nil
	}

	input := &service.SNSNotificationInput{
		Type:             req.Type,
		MessageID:        req.MessageId,
		TopicArn:         req.TopicArn,
		Message:          req.Message,
		SubscribeURL:     req.SubscribeUrl,
		Timestamp:        req.Timestamp,
		SignatureVersion: req.SignatureVersion,
		Signature:        req.Signature,
		SigningCertURL:   req.SigningCertUrl,
		Subject:          req.Subject,
		Token:            req.Token,
	}

	if err := s.inboundSvc.HandleSNSNotification(ctx, input); err != nil {
		return &emailapiv1.SNSNotificationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &emailapiv1.SNSNotificationResponse{
		Success: true,
		Message: "Inbound email notification processed successfully",
	}, nil
}
