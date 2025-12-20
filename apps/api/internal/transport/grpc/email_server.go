package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
)

// EmailServer implements the EmailService gRPC server.
type EmailServer struct {
	emailapiv1.UnimplementedEmailServiceServer
	svc *service.EmailService
}

// NewEmailServer creates a new EmailServer.
func NewEmailServer(svc *service.EmailService) *EmailServer {
	return &EmailServer{svc: svc}
}

// SendEmail handles the SendEmail RPC.
func (s *EmailServer) SendEmail(ctx context.Context, req *emailapiv1.SendEmailRequest) (*emailapiv1.SendEmailResponse, error) {
	// Get user ID from context (set by auth interceptor)
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	domainReq := &domain.SendEmailRequest{
		From:     req.From,
		To:       req.To,
		Cc:       req.Cc,
		Bcc:      req.Bcc,
		Subject:  req.Subject,
		Body:     req.Body,
		HTML:     req.Html,
		Metadata: convertMetadata(req.Metadata),
	}

	resp, err := s.svc.Send(ctx, userID, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send email: %v", err)
	}

	return &emailapiv1.SendEmailResponse{
		Id:     resp.ID,
		Status: toProtoEmailStatus(resp.Status),
	}, nil
}

// GetEmail handles the GetEmail RPC.
func (s *EmailServer) GetEmail(ctx context.Context, req *emailapiv1.GetEmailRequest) (*emailapiv1.Email, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	email, err := s.svc.GetByID(ctx, userID, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "email not found")
	}

	return toProtoEmail(email), nil
}

// ListEmails handles the ListEmails RPC.
func (s *EmailServer) ListEmails(ctx context.Context, req *emailapiv1.ListEmailsRequest) (*emailapiv1.ListEmailsResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	limit := int(req.Limit)
	offset := int(req.Offset)

	emails, err := s.svc.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list emails")
	}

	protoEmails := make([]*emailapiv1.Email, len(emails))
	for i, e := range emails {
		protoEmails[i] = toProtoEmail(e)
	}

	return &emailapiv1.ListEmailsResponse{
		Data:   protoEmails,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}

// Helper functions
func toProtoEmailStatus(s domain.EmailStatus) emailapiv1.EmailStatus {
	switch s {
	case domain.EmailStatusPending:
		return emailapiv1.EmailStatus_EMAIL_STATUS_PENDING
	case domain.EmailStatusSent:
		return emailapiv1.EmailStatus_EMAIL_STATUS_SENT
	case domain.EmailStatusDelivered:
		return emailapiv1.EmailStatus_EMAIL_STATUS_DELIVERED
	case domain.EmailStatusFailed:
		return emailapiv1.EmailStatus_EMAIL_STATUS_FAILED
	case domain.EmailStatusBounced:
		return emailapiv1.EmailStatus_EMAIL_STATUS_BOUNCED
	default:
		return emailapiv1.EmailStatus_EMAIL_STATUS_UNSPECIFIED
	}
}

func toProtoEmail(e *domain.Email) *emailapiv1.Email {
	return &emailapiv1.Email{
		Id:      e.ID,
		From:    e.From,
		To:      e.To,
		Cc:      e.Cc,
		Bcc:     e.Bcc,
		Subject: e.Subject,
		Body:    e.Body,
		Html:    e.HTML,
		Status:  toProtoEmailStatus(e.Status),
		UserId:  e.UserID,
	}
}

func convertMetadata(m map[string]string) domain.Metadata {
	if m == nil {
		return nil
	}
	result := make(domain.Metadata)
	for k, v := range m {
		result[k] = v
	}
	return result
}
