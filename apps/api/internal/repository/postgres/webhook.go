package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emailapi/api/internal/domain"
)

// WebhookRepository implements repository.WebhookRepository using PostgreSQL.
type WebhookRepository struct {
	pool *pgxpool.Pool
}

// Create stores a new webhook.
func (r *WebhookRepository) Create(ctx context.Context, webhook *domain.Webhook) error {
	query := `
		INSERT INTO webhooks (id, user_id, name, url, secret_hash, events, is_active, 
			retry_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	webhook.CreatedAt = now
	webhook.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		webhook.ID, webhook.UserID, webhook.Name, webhook.URL, webhook.Secret,
		webhook.Events, webhook.IsActive, webhook.RetryCount,
		webhook.CreatedAt, webhook.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	return nil
}

// GetByID retrieves a webhook by its ID.
func (r *WebhookRepository) GetByID(ctx context.Context, id string) (*domain.Webhook, error) {
	query := `
		SELECT id, user_id, name, url, events, is_active, retry_count,
			last_success, last_failure, created_at, updated_at
		FROM webhooks
		WHERE id = $1
	`

	var webhook domain.Webhook
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&webhook.ID, &webhook.UserID, &webhook.Name, &webhook.URL,
		&webhook.Events, &webhook.IsActive, &webhook.RetryCount,
		&webhook.LastSuccess, &webhook.LastFailure,
		&webhook.CreatedAt, &webhook.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	return &webhook, nil
}

// GetByUserID retrieves all webhooks for a user.
func (r *WebhookRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Webhook, error) {
	query := `
		SELECT id, user_id, name, url, events, is_active, retry_count,
			last_success, last_failure, created_at, updated_at
		FROM webhooks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*domain.Webhook
	for rows.Next() {
		var webhook domain.Webhook
		err := rows.Scan(
			&webhook.ID, &webhook.UserID, &webhook.Name, &webhook.URL,
			&webhook.Events, &webhook.IsActive, &webhook.RetryCount,
			&webhook.LastSuccess, &webhook.LastFailure,
			&webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

// GetActiveByEvent retrieves all active webhooks for a specific event type.
func (r *WebhookRepository) GetActiveByEvent(ctx context.Context, eventType domain.WebhookEventType) ([]*domain.Webhook, error) {
	query := `
		SELECT id, user_id, name, url, events, is_active, retry_count,
			last_success, last_failure, created_at, updated_at
		FROM webhooks
		WHERE is_active = true AND $1 = ANY(events)
	`

	rows, err := r.pool.Query(ctx, query, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhooks by event: %w", err)
	}
	defer rows.Close()

	var webhooks []*domain.Webhook
	for rows.Next() {
		var webhook domain.Webhook
		err := rows.Scan(
			&webhook.ID, &webhook.UserID, &webhook.Name, &webhook.URL,
			&webhook.Events, &webhook.IsActive, &webhook.RetryCount,
			&webhook.LastSuccess, &webhook.LastFailure,
			&webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

// Update updates an existing webhook.
func (r *WebhookRepository) Update(ctx context.Context, webhook *domain.Webhook) error {
	query := `
		UPDATE webhooks
		SET name = $2, url = $3, events = $4, is_active = $5, retry_count = $6,
			last_success = $7, last_failure = $8, updated_at = $9
		WHERE id = $1
	`

	webhook.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		webhook.ID, webhook.Name, webhook.URL, webhook.Events, webhook.IsActive,
		webhook.RetryCount, webhook.LastSuccess, webhook.LastFailure, webhook.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	return nil
}

// Delete removes a webhook.
func (r *WebhookRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM webhooks WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}

// CreateDelivery stores a webhook delivery attempt.
func (r *WebhookRepository) CreateDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (id, webhook_id, event_type, payload, 
			response_code, response_body, success, attempt_count, next_retry, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	delivery.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		delivery.ID, delivery.WebhookID, delivery.EventType, delivery.Payload,
		delivery.ResponseCode, delivery.ResponseBody, delivery.Success,
		delivery.AttemptCount, delivery.NextRetry, delivery.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create webhook delivery: %w", err)
	}

	return nil
}

// GetDeliveriesByWebhookID retrieves delivery history for a webhook.
func (r *WebhookRepository) GetDeliveriesByWebhookID(ctx context.Context, webhookID string, limit int) ([]*domain.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_type, payload, response_code, response_body,
			success, attempt_count, next_retry, created_at
		FROM webhook_deliveries
		WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, webhookID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhook deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*domain.WebhookDelivery
	for rows.Next() {
		var delivery domain.WebhookDelivery
		err := rows.Scan(
			&delivery.ID, &delivery.WebhookID, &delivery.EventType, &delivery.Payload,
			&delivery.ResponseCode, &delivery.ResponseBody, &delivery.Success,
			&delivery.AttemptCount, &delivery.NextRetry, &delivery.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook delivery: %w", err)
		}
		deliveries = append(deliveries, &delivery)
	}

	return deliveries, nil
}
