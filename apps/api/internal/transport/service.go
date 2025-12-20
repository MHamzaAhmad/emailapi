package transport

import (
	"github.com/emailapi/api/internal/domain"
)

// Service is the single interface that the Transport layer uses
// to interact with business logic.
type Service interface {
	// Email operations
	SendEmail(userID string, req *domain.SendEmailRequest) (*domain.SendEmailResponse, error)
	GetEmail(userID, emailID string) (*domain.Email, error)
	ListEmails(userID string, limit, offset int) ([]*domain.Email, error)

	// User operations
	GetUser(id string) (*domain.User, error)
	GetUserByAPIKey(apiKey string) (*domain.User, error)

	// Webhook operations
	CreateWebhook(userID string, req *domain.CreateWebhookRequest) (*domain.Webhook, string, error)
	GetWebhook(userID, webhookID string) (*domain.Webhook, error)
	ListWebhooks(userID string) ([]*domain.Webhook, error)
	UpdateWebhook(userID, webhookID string, req *domain.UpdateWebhookRequest) (*domain.Webhook, error)
	DeleteWebhook(userID, webhookID string) error
}
