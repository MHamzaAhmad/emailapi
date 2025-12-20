package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/domain"
)

// WebhookRepository implements repository.WebhookRepository using sqlc-generated queries.
type WebhookRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewWebhookRepository creates a new WebhookRepository.
func NewWebhookRepository(pool *pgxpool.Pool) *WebhookRepository {
	return &WebhookRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Create stores a new webhook using sqlc.
func (r *WebhookRepository) Create(ctx context.Context, webhook *domain.Webhook) error {
	result, err := r.queries.CreateWebhook(ctx, db.CreateWebhookParams{
		ID:         webhook.ID,
		UserID:     webhook.UserID,
		Name:       webhook.Name,
		Url:        webhook.URL,
		SecretHash: webhook.Secret,
		Events:     toStringArray(webhook.Events),
		IsActive:   webhook.IsActive,
		RetryCount: int32(webhook.RetryCount),
		CreatedAt:  toPgTimestampNow(),
		UpdatedAt:  toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	webhook.CreatedAt = result.CreatedAt.Time
	webhook.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// GetByID retrieves a webhook by its ID using sqlc.
func (r *WebhookRepository) GetByID(ctx context.Context, id string) (*domain.Webhook, error) {
	row, err := r.queries.GetWebhookByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	return toDomainWebhookFromRow(row), nil
}

// GetByUserID retrieves all webhooks for a user using sqlc.
func (r *WebhookRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Webhook, error) {
	rows, err := r.queries.GetWebhooksByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %w", err)
	}

	webhooks := make([]*domain.Webhook, len(rows))
	for i, row := range rows {
		webhooks[i] = toDomainWebhookFromRow(row)
	}
	return webhooks, nil
}

// GetActiveByEvent retrieves all active webhooks for a specific event type using sqlc.
func (r *WebhookRepository) GetActiveByEvent(ctx context.Context, eventType domain.WebhookEventType) ([]*domain.Webhook, error) {
	rows, err := r.queries.GetActiveWebhooksByEvent(ctx, string(eventType))
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks by event: %w", err)
	}

	webhooks := make([]*domain.Webhook, len(rows))
	for i, row := range rows {
		webhooks[i] = toDomainWebhookFromRow(row)
	}
	return webhooks, nil
}

// Update updates an existing webhook using sqlc.
func (r *WebhookRepository) Update(ctx context.Context, webhook *domain.Webhook) error {
	result, err := r.queries.UpdateWebhook(ctx, db.UpdateWebhookParams{
		ID:          webhook.ID,
		Name:        webhook.Name,
		Url:         webhook.URL,
		Events:      toStringArray(webhook.Events),
		IsActive:    webhook.IsActive,
		RetryCount:  int32(webhook.RetryCount),
		LastSuccess: toPgTimestamp(webhook.LastSuccess),
		LastFailure: toPgTimestamp(webhook.LastFailure),
		UpdatedAt:   toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	webhook.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// Delete removes a webhook using sqlc.
func (r *WebhookRepository) Delete(ctx context.Context, id string) error {
	err := r.queries.DeleteWebhook(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	return nil
}

// CreateDelivery stores a webhook delivery attempt using sqlc.
func (r *WebhookRepository) CreateDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error {
	result, err := r.queries.CreateWebhookDelivery(ctx, db.CreateWebhookDeliveryParams{
		ID:           delivery.ID,
		WebhookID:    delivery.WebhookID,
		EventType:    string(delivery.EventType),
		Payload:      delivery.Payload,
		ResponseCode: toPgInt4(delivery.ResponseCode),
		ResponseBody: toPgTextPtr(delivery.ResponseBody),
		Success:      delivery.Success,
		AttemptCount: int32(delivery.AttemptCount),
		NextRetry:    toPgTimestamp(delivery.NextRetry),
		CreatedAt:    toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to create webhook delivery: %w", err)
	}

	delivery.CreatedAt = result.CreatedAt.Time
	return nil
}

// GetDeliveriesByWebhookID retrieves delivery history for a webhook using sqlc.
func (r *WebhookRepository) GetDeliveriesByWebhookID(ctx context.Context, webhookID string, limit int) ([]*domain.WebhookDelivery, error) {
	rows, err := r.queries.GetWebhookDeliveriesByWebhookID(ctx, db.GetWebhookDeliveriesByWebhookIDParams{
		WebhookID: webhookID,
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query webhook deliveries: %w", err)
	}

	deliveries := make([]*domain.WebhookDelivery, len(rows))
	for i, row := range rows {
		deliveries[i] = &domain.WebhookDelivery{
			ID:           row.ID,
			WebhookID:    row.WebhookID,
			EventType:    domain.WebhookEventType(row.EventType),
			Payload:      row.Payload,
			ResponseCode: fromPgInt4(row.ResponseCode),
			ResponseBody: fromPgTextPtr(row.ResponseBody),
			Success:      row.Success,
			AttemptCount: int(row.AttemptCount),
			NextRetry:    fromPgTimestamp(row.NextRetry),
			CreatedAt:    row.CreatedAt.Time,
		}
	}
	return deliveries, nil
}

// toDomainWebhookFromRow converts a sqlc webhook row to domain.Webhook.
func toDomainWebhookFromRow(row db.GetWebhookByIDRow) *domain.Webhook {
	return &domain.Webhook{
		ID:          row.ID,
		UserID:      row.UserID,
		Name:        row.Name,
		URL:         row.Url,
		Events:      toWebhookEventTypes(row.Events),
		IsActive:    row.IsActive,
		RetryCount:  int(row.RetryCount),
		LastSuccess: fromPgTimestamp(row.LastSuccess),
		LastFailure: fromPgTimestamp(row.LastFailure),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

// toStringArray converts []WebhookEventType to []string.
func toStringArray(events []domain.WebhookEventType) []string {
	result := make([]string, len(events))
	for i, e := range events {
		result[i] = string(e)
	}
	return result
}

// toWebhookEventTypes converts []string to []WebhookEventType.
func toWebhookEventTypes(events []string) []domain.WebhookEventType {
	result := make([]domain.WebhookEventType, len(events))
	for i, e := range events {
		result[i] = domain.WebhookEventType(e)
	}
	return result
}
