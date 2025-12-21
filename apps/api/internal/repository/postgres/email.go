package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/domain"
)

// EmailRepository implements email data access using sqlc-generated queries.
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

// Create stores a new email.
func (r *EmailRepository) Create(ctx context.Context, email *domain.Email) error {
	metadataJSON, err := json.Marshal(email.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	result, err := r.queries.CreateEmail(ctx, db.CreateEmailParams{
		ID:           email.ID,
		UserID:       email.UserID,
		FromAddress:  email.From,
		ToAddresses:  email.To,
		CcAddresses:  email.Cc,
		BccAddresses: email.Bcc,
		Subject:      email.Subject,
		Body:         toPgText(email.Body),
		Html:         toPgText(email.HTML),
		Status:       string(email.Status),
		Metadata:     metadataJSON,
		ScheduledAt:  toPgTimestamp(email.ScheduledAt),
	})
	if err != nil {
		return fmt.Errorf("failed to create email: %w", err)
	}

	email.CreatedAt = result.CreatedAt.Time
	email.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// GetByID retrieves an email by its ID.
func (r *EmailRepository) GetByID(ctx context.Context, id string) (*domain.Email, error) {
	row, err := r.queries.GetEmailByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}
	return dbEmailToDomain(row), nil
}

// GetByUserID retrieves emails for a user with pagination.
func (r *EmailRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Email, error) {
	rows, err := r.queries.GetEmailsByUserID(ctx, db.GetEmailsByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}

	emails := make([]*domain.Email, len(rows))
	for i, row := range rows {
		emails[i] = dbEmailToDomain(row)
	}
	return emails, nil
}

// UpdateStatus updates the status of an email.
func (r *EmailRepository) UpdateStatus(ctx context.Context, id string, status domain.EmailStatus, errorMessage string) error {
	_, err := r.queries.UpdateEmailStatus(ctx, db.UpdateEmailStatusParams{
		ID:           id,
		Status:       string(status),
		ErrorMessage: toPgText(errorMessage),
	})
	if err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}
	return nil
}

// UpdateSent marks an email as sent with the provider ID.
func (r *EmailRepository) UpdateSent(ctx context.Context, id, providerID string) error {
	_, err := r.queries.UpdateEmailSent(ctx, db.UpdateEmailSentParams{
		ID:         id,
		ProviderID: toPgText(providerID),
	})
	if err != nil {
		return fmt.Errorf("failed to update email sent: %w", err)
	}
	return nil
}

// Delete removes an email.
func (r *EmailRepository) Delete(ctx context.Context, id string) error {
	if err := r.queries.DeleteEmail(ctx, id); err != nil {
		return fmt.Errorf("failed to delete email: %w", err)
	}
	return nil
}

// CreateAttachment stores a new email attachment.
func (r *EmailRepository) CreateAttachment(ctx context.Context, att *domain.EmailAttachment) error {
	result, err := r.queries.CreateEmailAttachment(ctx, db.CreateEmailAttachmentParams{
		ID:          att.ID,
		EmailID:     att.EmailID,
		Filename:    att.Filename,
		ContentType: att.ContentType,
		S3Key:       att.S3Key,
		SizeBytes:   att.SizeBytes,
		ScanStatus:  string(att.ScanStatus),
	})
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	att.CreatedAt = result.CreatedAt.Time
	return nil
}

// GetAttachmentsByEmailID retrieves all attachments for an email.
func (r *EmailRepository) GetAttachmentsByEmailID(ctx context.Context, emailID string) ([]*domain.EmailAttachment, error) {
	rows, err := r.queries.GetAttachmentsByEmailID(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}

	attachments := make([]*domain.EmailAttachment, len(rows))
	for i, row := range rows {
		attachments[i] = dbAttachmentToDomain(row)
	}
	return attachments, nil
}

// GetAttachmentByS3Key retrieves an attachment by its S3 key.
func (r *EmailRepository) GetAttachmentByS3Key(ctx context.Context, s3Key string) (*domain.EmailAttachment, error) {
	row, err := r.queries.GetAttachmentByS3Key(ctx, s3Key)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment by S3 key: %w", err)
	}
	return dbAttachmentToDomain(row), nil
}

// UpdateAttachmentScanStatus updates the scan status of an attachment.
func (r *EmailRepository) UpdateAttachmentScanStatus(ctx context.Context, id string, scanStatus domain.AttachmentScanStatus) error {
	_, err := r.queries.UpdateAttachmentScanStatus(ctx, db.UpdateAttachmentScanStatusParams{
		ID:         id,
		ScanStatus: string(scanStatus),
	})
	if err != nil {
		return fmt.Errorf("failed to update attachment scan status: %w", err)
	}
	return nil
}

// CheckAllAttachmentsScanned checks if all attachments for an email have been scanned.
// Returns: allScanned, allClean, hasThreats
func (r *EmailRepository) CheckAllAttachmentsScanned(ctx context.Context, emailID string) (bool, bool, bool, error) {
	result, err := r.queries.CheckAllAttachmentsScanned(ctx, emailID)
	if err != nil {
		return false, false, false, fmt.Errorf("failed to check attachment scan status: %w", err)
	}
	return result.AllScanned, result.AllClean, result.HasThreats, nil
}

// dbEmailToDomain converts a sqlc Email to domain.Email.
func dbEmailToDomain(row db.Email) *domain.Email {
	var metadata domain.Metadata
	if row.Metadata != nil {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}

	return &domain.Email{
		ID:           row.ID,
		UserID:       row.UserID,
		From:         row.FromAddress,
		To:           row.ToAddresses,
		Cc:           row.CcAddresses,
		Bcc:          row.BccAddresses,
		Subject:      row.Subject,
		Body:         fromPgText(row.Body),
		HTML:         fromPgText(row.Html),
		Status:       domain.EmailStatus(row.Status),
		ProviderID:   fromPgText(row.ProviderID),
		Metadata:     metadata,
		ErrorMessage: fromPgText(row.ErrorMessage),
		ScheduledAt:  fromPgTimestamp(row.ScheduledAt),
		SentAt:       fromPgTimestamp(row.SentAt),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}

// dbAttachmentToDomain converts a sqlc EmailAttachment to domain.EmailAttachment.
func dbAttachmentToDomain(row db.EmailAttachment) *domain.EmailAttachment {
	return &domain.EmailAttachment{
		ID:            row.ID,
		EmailID:       row.EmailID,
		Filename:      row.Filename,
		ContentType:   row.ContentType,
		S3Key:         row.S3Key,
		SizeBytes:     row.SizeBytes,
		ScanStatus:    domain.AttachmentScanStatus(row.ScanStatus),
		ScanCheckedAt: fromPgTimestamp(row.ScanCheckedAt),
		CreatedAt:     row.CreatedAt.Time,
	}
}
