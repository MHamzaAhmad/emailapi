package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emailapi/api/internal/domain"
)

// EmailRepository implements repository.EmailRepository using PostgreSQL.
type EmailRepository struct {
	pool *pgxpool.Pool
}

// Create stores a new email.
func (r *EmailRepository) Create(ctx context.Context, email *domain.Email) error {
	query := `
		INSERT INTO emails (id, from_address, to_addresses, cc_addresses, bcc_addresses, 
			subject, body, html_body, status, provider_id, user_id, webhook_id, 
			metadata, scheduled_at, sent_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	now := time.Now()
	email.CreatedAt = now
	email.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		email.ID, email.From, email.To, email.Cc, email.Bcc,
		email.Subject, email.Body, email.HTML, email.Status,
		email.ProviderID, email.UserID, email.WebhookID, email.Metadata,
		email.ScheduledAt, email.SentAt, email.CreatedAt, email.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create email: %w", err)
	}

	return nil
}

// GetByID retrieves an email by its ID.
func (r *EmailRepository) GetByID(ctx context.Context, id string) (*domain.Email, error) {
	query := `
		SELECT id, from_address, to_addresses, cc_addresses, bcc_addresses,
			subject, body, html_body, status, provider_id, user_id, webhook_id,
			metadata, scheduled_at, sent_at, created_at, updated_at
		FROM emails
		WHERE id = $1
	`

	var email domain.Email
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&email.ID, &email.From, &email.To, &email.Cc, &email.Bcc,
		&email.Subject, &email.Body, &email.HTML, &email.Status,
		&email.ProviderID, &email.UserID, &email.WebhookID, &email.Metadata,
		&email.ScheduledAt, &email.SentAt, &email.CreatedAt, &email.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	return &email, nil
}

// GetByUserID retrieves all emails for a user.
func (r *EmailRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Email, error) {
	query := `
		SELECT id, from_address, to_addresses, cc_addresses, bcc_addresses,
			subject, body, html_body, status, provider_id, user_id, webhook_id,
			metadata, scheduled_at, sent_at, created_at, updated_at
		FROM emails
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query emails: %w", err)
	}
	defer rows.Close()

	var emails []*domain.Email
	for rows.Next() {
		var email domain.Email
		err := rows.Scan(
			&email.ID, &email.From, &email.To, &email.Cc, &email.Bcc,
			&email.Subject, &email.Body, &email.HTML, &email.Status,
			&email.ProviderID, &email.UserID, &email.WebhookID, &email.Metadata,
			&email.ScheduledAt, &email.SentAt, &email.CreatedAt, &email.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}
		emails = append(emails, &email)
	}

	return emails, nil
}

// Update updates an existing email.
func (r *EmailRepository) Update(ctx context.Context, email *domain.Email) error {
	query := `
		UPDATE emails
		SET from_address = $2, to_addresses = $3, cc_addresses = $4, bcc_addresses = $5,
			subject = $6, body = $7, html_body = $8, status = $9, provider_id = $10,
			webhook_id = $11, metadata = $12, scheduled_at = $13, sent_at = $14, updated_at = $15
		WHERE id = $1
	`

	email.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		email.ID, email.From, email.To, email.Cc, email.Bcc,
		email.Subject, email.Body, email.HTML, email.Status, email.ProviderID,
		email.WebhookID, email.Metadata, email.ScheduledAt, email.SentAt, email.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	return nil
}

// UpdateStatus updates only the status of an email.
func (r *EmailRepository) UpdateStatus(ctx context.Context, id string, status domain.EmailStatus) error {
	query := `UPDATE emails SET status = $2, updated_at = $3 WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}

	return nil
}
