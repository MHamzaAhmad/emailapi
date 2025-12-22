package grpc

import (
	"context"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/service"
)

// SnsServer wraps SNSNotificationService for gRPC.
// These endpoints are public but authenticated via SNS signature verification.
type SnsServer struct {
	emailapiv1.UnimplementedSnsServiceServer
	svc *service.SNSNotificationService
}

// NewSnsServer creates a new SnsServer.
func NewSnsServer(svc *service.SNSNotificationService) *SnsServer {
	return &SnsServer{
		svc: svc,
	}
}

// HandleSNSNotification processes SNS notifications.
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
