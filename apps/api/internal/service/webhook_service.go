package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"github.com/emailapi/api/internal/domain"
)

// WebhookService handles webhook business logic.
type WebhookService struct {
	store Store
}

// NewWebhookService creates a new WebhookService.
func NewWebhookService(store Store) *WebhookService {
	return &WebhookService{store: store}
}

// Create creates a new webhook.
func (s *WebhookService) Create(ctx context.Context, userID string, req *domain.CreateWebhookRequest) (*domain.Webhook, string, error) {
	// Generate webhook secret
	secret, hashedSecret, err := s.generateSecret()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate secret: %w", err)
	}

	webhook := &domain.Webhook{
		ID:         uuid.New().String(),
		UserID:     userID,
		Name:       req.Name,
		URL:        req.URL,
		Secret:     hashedSecret,
		Events:     req.Events,
		IsActive:   true,
		RetryCount: 3,
	}

	if err := s.store.Webhooks().Create(ctx, webhook); err != nil {
		return nil, "", fmt.Errorf("failed to create webhook: %w", err)
	}

	return webhook, secret, nil
}

// GetByID retrieves a webhook by ID.
func (s *WebhookService) GetByID(ctx context.Context, userID, webhookID string) (*domain.Webhook, error) {
	webhook, err := s.store.Webhooks().GetByID(ctx, webhookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	// Authorization check
	if webhook.UserID != userID {
		return nil, fmt.Errorf("webhook not found")
	}

	return webhook, nil
}

// List retrieves all webhooks for a user.
func (s *WebhookService) List(ctx context.Context, userID string) ([]*domain.Webhook, error) {
	return s.store.Webhooks().GetByUserID(ctx, userID)
}

// Update updates a webhook.
func (s *WebhookService) Update(ctx context.Context, userID, webhookID string, req *domain.UpdateWebhookRequest) (*domain.Webhook, error) {
	webhook, err := s.GetByID(ctx, userID, webhookID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		webhook.Name = *req.Name
	}
	if req.URL != nil {
		webhook.URL = *req.URL
	}
	if req.Events != nil {
		webhook.Events = req.Events
	}
	if req.IsActive != nil {
		webhook.IsActive = *req.IsActive
	}

	if err := s.store.Webhooks().Update(ctx, webhook); err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return webhook, nil
}

// Delete deletes a webhook.
func (s *WebhookService) Delete(ctx context.Context, userID, webhookID string) error {
	// Verify ownership
	_, err := s.GetByID(ctx, userID, webhookID)
	if err != nil {
		return err
	}

	return s.store.Webhooks().Delete(ctx, webhookID)
}

// generateSecret generates a webhook secret.
func (s *WebhookService) generateSecret() (raw, hashed string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}

	raw = "whsec_" + hex.EncodeToString(bytes)
	// In production, you'd hash this
	hashed = raw

	return raw, hashed, nil
}
