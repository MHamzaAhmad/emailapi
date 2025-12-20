package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/domain"
)

// EmailRepository implements repository.EmailRepository using sqlc-generated queries.
type EmailRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewEmailRepository creates a new EmailRepository.
func NewEmailRepository(pool *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Create stores a new email using sqlc.
func (r *EmailRepository) Create(ctx context.Context, email *domain.Email) error {
	metadata, err := json.Marshal(email.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.queries.CreateEmail(ctx, db.CreateEmailParams{
		ID:           email.ID,
		FromAddress:  email.From,
		ToAddresses:  email.To,
		CcAddresses:  email.Cc,
		BccAddresses: email.Bcc,
		Subject:      email.Subject,
		Body:         toPgText(email.Body),
		HtmlBody:     toPgText(email.HTML),
		Status:       string(email.Status),
		ProviderID:   toPgText(email.ProviderID),
		UserID:       email.UserID,
		WebhookID:    toPgText(email.WebhookID),
		Metadata:     metadata,
		ScheduledAt:  toPgTimestamp(email.ScheduledAt),
		SentAt:       toPgTimestamp(email.SentAt),
		CreatedAt:    toPgTimestampNow(),
		UpdatedAt:    toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to create email: %w", err)
	}

	email.CreatedAt = result.CreatedAt.Time
	email.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// GetByID retrieves an email by its ID using sqlc.
func (r *EmailRepository) GetByID(ctx context.Context, id string) (*domain.Email, error) {
	row, err := r.queries.GetEmailByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}
	return toDomainEmail(row), nil
}

// GetByUserID retrieves all emails for a user using sqlc.
func (r *EmailRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Email, error) {
	rows, err := r.queries.GetEmailsByUserID(ctx, db.GetEmailsByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query emails: %w", err)
	}

	emails := make([]*domain.Email, len(rows))
	for i, row := range rows {
		emails[i] = toDomainEmail(row)
	}
	return emails, nil
}

// Update updates an existing email using sqlc.
func (r *EmailRepository) Update(ctx context.Context, email *domain.Email) error {
	metadata, err := json.Marshal(email.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := r.queries.UpdateEmail(ctx, db.UpdateEmailParams{
		ID:           email.ID,
		FromAddress:  email.From,
		ToAddresses:  email.To,
		CcAddresses:  email.Cc,
		BccAddresses: email.Bcc,
		Subject:      email.Subject,
		Body:         toPgText(email.Body),
		HtmlBody:     toPgText(email.HTML),
		Status:       string(email.Status),
		ProviderID:   toPgTextPtr(email.ProviderID),
		WebhookID:    toPgTextPtr(email.WebhookID),
		Metadata:     metadata,
		ScheduledAt:  toPgTimestamp(email.ScheduledAt),
		SentAt:       toPgTimestamp(email.SentAt),
		UpdatedAt:    toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	email.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// UpdateStatus updates only the status of an email using sqlc.
func (r *EmailRepository) UpdateStatus(ctx context.Context, id string, status domain.EmailStatus) error {
	err := r.queries.UpdateEmailStatus(ctx, db.UpdateEmailStatusParams{
		ID:     id,
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}
	return nil
}

// toDomainEmail converts a sqlc Email to a domain Email.
func toDomainEmail(e db.Email) *domain.Email {
	var metadata map[string]string
	if e.Metadata != nil {
		_ = json.Unmarshal(e.Metadata, &metadata)
	}

	return &domain.Email{
		ID:          e.ID,
		From:        e.FromAddress,
		To:          e.ToAddresses,
		Cc:          e.CcAddresses,
		Bcc:         e.BccAddresses,
		Subject:     e.Subject,
		Body:        fromPgText(e.Body),
		HTML:        fromPgText(e.HtmlBody),
		Status:      domain.EmailStatus(e.Status),
		ProviderID:  fromPgText(e.ProviderID),
		UserID:      e.UserID,
		WebhookID:   fromPgText(e.WebhookID),
		Metadata:    toMetadata(metadata),
		ScheduledAt: fromPgTimestamp(e.ScheduledAt),
		SentAt:      fromPgTimestamp(e.SentAt),
		CreatedAt:   e.CreatedAt.Time,
		UpdatedAt:   e.UpdatedAt.Time,
	}
}
