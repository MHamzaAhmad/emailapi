package connect

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/gen/v1/v1connect"
	"github.com/emailapi/api/internal/service"
)

// SnsHandler implements the Connect SnsServiceHandler.
// These endpoints are public but authenticated via SNS signature verification.
type SnsHandler struct {
	v1connect.UnimplementedSnsServiceHandler
	svc        *service.SNSNotificationService
	inboundSvc *service.InboundEmailService
}

// NewSnsHandler creates a new SnsHandler.
func NewSnsHandler(svc *service.SNSNotificationService, inboundSvc *service.InboundEmailService) *SnsHandler {
	return &SnsHandler{
		svc:        svc,
		inboundSvc: inboundSvc,
	}
}

// HandleSNSNotification processes SNS notifications for outbound email events.
func (h *SnsHandler) HandleSNSNotification(
	ctx context.Context,
	req *connect.Request[v1.SNSNotificationRequest],
) (*connect.Response[v1.SNSNotificationResponse], error) {
	input := &service.SNSInput{
		Type:             req.Msg.Type,
		MessageID:        req.Msg.MessageId,
		TopicArn:         req.Msg.TopicArn,
		Message:          req.Msg.Message,
		SubscribeURL:     req.Msg.SubscribeUrl,
		Timestamp:        req.Msg.Timestamp,
		SignatureVersion: req.Msg.SignatureVersion,
		Signature:        req.Msg.Signature,
		SigningCertURL:   req.Msg.SigningCertUrl,
		Subject:          req.Msg.Subject,
		Token:            req.Msg.Token,
	}

	if err := h.svc.HandleNotification(ctx, input); err != nil {
		return connect.NewResponse(&v1.SNSNotificationResponse{
			Success: false,
			Message: err.Error(),
		}), nil
	}

	return connect.NewResponse(&v1.SNSNotificationResponse{
		Success: true,
		Message: "SNS notification processed successfully",
	}), nil
}

// HandleInboundSNSNotification processes SNS notifications for inbound emails (replies).
func (h *SnsHandler) HandleInboundSNSNotification(
	ctx context.Context,
	req *connect.Request[v1.SNSNotificationRequest],
) (*connect.Response[v1.SNSNotificationResponse], error) {
	if h.inboundSvc == nil {
		return connect.NewResponse(&v1.SNSNotificationResponse{
			Success: false,
			Message: "Inbound email service not configured",
		}), nil
	}

	input := &service.SNSNotificationInput{
		Type:             req.Msg.Type,
		MessageID:        req.Msg.MessageId,
		TopicArn:         req.Msg.TopicArn,
		Message:          req.Msg.Message,
		SubscribeURL:     req.Msg.SubscribeUrl,
		Timestamp:        req.Msg.Timestamp,
		SignatureVersion: req.Msg.SignatureVersion,
		Signature:        req.Msg.Signature,
		SigningCertURL:   req.Msg.SigningCertUrl,
		Subject:          req.Msg.Subject,
		Token:            req.Msg.Token,
	}

	if err := h.inboundSvc.HandleSNSNotification(ctx, input); err != nil {
		return connect.NewResponse(&v1.SNSNotificationResponse{
			Success: false,
			Message: err.Error(),
		}), nil
	}

	return connect.NewResponse(&v1.SNSNotificationResponse{
		Success: true,
		Message: "Inbound email notification processed successfully",
	}), nil
}
