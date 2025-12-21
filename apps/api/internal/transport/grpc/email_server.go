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

func (s *EmailServer) GetEmail(ctx context.Context, req *emailapiv1.GetEmailRequest) (*emailapiv1.Email, error) {
	// Not implemented yet in service, but required by interface
	// return s.svc.GetEmail(ctx, req)
	return nil, nil
}

func (s *EmailServer) ListEmails(ctx context.Context, req *emailapiv1.ListEmailsRequest) (*emailapiv1.ListEmailsResponse, error) {
	// Not implemented yet in service
	// return s.svc.ListEmails(ctx, req)
	return nil, nil
}
