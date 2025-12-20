package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/service"
)

// WebhookServer implements the WebhookService gRPC server.
type WebhookServer struct {
	emailapiv1.UnimplementedWebhookServiceServer
	svc *service.WebhookService
}

// NewWebhookServer creates a new WebhookServer.
func NewWebhookServer(svc *service.WebhookService) *WebhookServer {
	return &WebhookServer{svc: svc}
}

// CreateWebhook handles the CreateWebhook RPC.
func (s *WebhookServer) CreateWebhook(ctx context.Context, req *emailapiv1.CreateWebhookRequest) (*emailapiv1.CreateWebhookResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	domainReq := &domain.CreateWebhookRequest{
		Name:   req.Name,
		URL:    req.Url,
		Events: toDomainEvents(req.Events),
	}

	webhook, secret, err := s.svc.Create(ctx, userID, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create webhook: %v", err)
	}

	return &emailapiv1.CreateWebhookResponse{
		Webhook: toProtoWebhook(webhook),
		Secret:  secret,
		Message: "Store this secret securely. It will not be shown again.",
	}, nil
}

// GetWebhook handles the GetWebhook RPC.
func (s *WebhookServer) GetWebhook(ctx context.Context, req *emailapiv1.GetWebhookRequest) (*emailapiv1.Webhook, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	webhook, err := s.svc.GetByID(ctx, userID, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "webhook not found")
	}

	return toProtoWebhook(webhook), nil
}

// ListWebhooks handles the ListWebhooks RPC.
func (s *WebhookServer) ListWebhooks(ctx context.Context, req *emailapiv1.ListWebhooksRequest) (*emailapiv1.ListWebhooksResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	webhooks, err := s.svc.List(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list webhooks")
	}

	protoWebhooks := make([]*emailapiv1.Webhook, len(webhooks))
	for i, w := range webhooks {
		protoWebhooks[i] = toProtoWebhook(w)
	}

	return &emailapiv1.ListWebhooksResponse{
		Data: protoWebhooks,
	}, nil
}

// UpdateWebhook handles the UpdateWebhook RPC.
func (s *WebhookServer) UpdateWebhook(ctx context.Context, req *emailapiv1.UpdateWebhookRequest) (*emailapiv1.Webhook, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	domainReq := &domain.UpdateWebhookRequest{}
	if req.Name != nil {
		domainReq.Name = req.Name
	}
	if req.Url != nil {
		domainReq.URL = req.Url
	}
	if len(req.Events) > 0 {
		domainReq.Events = toDomainEvents(req.Events)
	}
	if req.IsActive != nil {
		domainReq.IsActive = req.IsActive
	}

	webhook, err := s.svc.Update(ctx, userID, req.Id, domainReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update webhook: %v", err)
	}

	return toProtoWebhook(webhook), nil
}

// DeleteWebhook handles the DeleteWebhook RPC.
func (s *WebhookServer) DeleteWebhook(ctx context.Context, req *emailapiv1.DeleteWebhookRequest) (*emailapiv1.DeleteWebhookResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if err := s.svc.Delete(ctx, userID, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete webhook: %v", err)
	}

	return &emailapiv1.DeleteWebhookResponse{}, nil
}

// Helper functions
func toDomainEvents(events []emailapiv1.WebhookEventType) []domain.WebhookEventType {
	result := make([]domain.WebhookEventType, len(events))
	for i, e := range events {
		result[i] = toDomainEvent(e)
	}
	return result
}

func toDomainEvent(e emailapiv1.WebhookEventType) domain.WebhookEventType {
	switch e {
	case emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_SENT:
		return domain.WebhookEventEmailSent
	case emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_DELIVERED:
		return domain.WebhookEventEmailDelivered
	case emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_FAILED:
		return domain.WebhookEventEmailFailed
	case emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_BOUNCED:
		return domain.WebhookEventEmailBounced
	case emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_OPENED:
		return domain.WebhookEventEmailOpened
	case emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_CLICKED:
		return domain.WebhookEventEmailClicked
	default:
		return ""
	}
}

func toProtoWebhook(w *domain.Webhook) *emailapiv1.Webhook {
	return &emailapiv1.Webhook{
		Id:         w.ID,
		UserId:     w.UserID,
		Name:       w.Name,
		Url:        w.URL,
		Events:     toProtoEvents(w.Events),
		IsActive:   w.IsActive,
		RetryCount: int32(w.RetryCount),
	}
}

func toProtoEvents(events []domain.WebhookEventType) []emailapiv1.WebhookEventType {
	result := make([]emailapiv1.WebhookEventType, len(events))
	for i, e := range events {
		result[i] = toProtoEvent(e)
	}
	return result
}

func toProtoEvent(e domain.WebhookEventType) emailapiv1.WebhookEventType {
	switch e {
	case domain.WebhookEventEmailSent:
		return emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_SENT
	case domain.WebhookEventEmailDelivered:
		return emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_DELIVERED
	case domain.WebhookEventEmailFailed:
		return emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_FAILED
	case domain.WebhookEventEmailBounced:
		return emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_BOUNCED
	case domain.WebhookEventEmailOpened:
		return emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_OPENED
	case domain.WebhookEventEmailClicked:
		return emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_EMAIL_CLICKED
	default:
		return emailapiv1.WebhookEventType_WEBHOOK_EVENT_TYPE_UNSPECIFIED
	}
}
