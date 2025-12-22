package grpc

import (
	"context"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/service"
)

type EmailServer struct {
	emailapiv1.UnimplementedEmailServiceServer
	svc *service.EmailService
}

func NewEmailServer(svc *service.EmailService) *EmailServer {
	return &EmailServer{svc: svc}
}

func (s *EmailServer) SendEmail(ctx context.Context, req *emailapiv1.SendEmailRequest) (*emailapiv1.SendEmailResponse, error) {
	return s.svc.SendEmail(ctx, req)
}
