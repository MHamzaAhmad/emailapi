package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/domain"
)

// DomainRepository implements repository.DomainRepository using PostgreSQL and sqlc.
type DomainRepository struct {
	queries *db.Queries
}

// NewDomainRepository creates a new DomainRepository.
func NewDomainRepository(queries *db.Queries) *DomainRepository {
	return &DomainRepository{queries: queries}
}

// Create stores a new sending domain.
func (r *DomainRepository) Create(ctx context.Context, d *domain.SendingDomain) error {
	err := r.queries.CreateDomain(ctx, db.CreateDomainParams{
		ID:                 d.ID,
		UserID:             d.UserID,
		DomainName:         d.Domain,
		Status:             string(d.Status),
		VerifiedForSending: d.VerifiedForSending,
		DkimTokens:         d.DkimTokens,
		DkimStatus:         string(d.DkimStatus),
		MailFromDomain:     toPgText(d.MailFromDomain),
		MailFromStatus:     toPgTextFromStatus(d.MailFromStatus),
		Region:             d.Region,
		CreatedAt:          toPgTimestampFromTime(d.CreatedAt),
		UpdatedAt:          toPgTimestampFromTime(d.UpdatedAt),
		LastVerifiedAt:     toPgTimestampFromTimePtr(d.LastCheckedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to create domain: %w", err)
	}
	return nil
}

// GetByID retrieves a domain by its ID.
func (r *DomainRepository) GetByID(ctx context.Context, id string) (*domain.SendingDomain, error) {
	row, err := r.queries.GetDomainByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}
	return dbDomainToModel(row), nil
}

// GetByDomainName retrieves a domain by its name for a specific user.
func (r *DomainRepository) GetByDomainName(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error) {
	row, err := r.queries.GetDomainByName(ctx, db.GetDomainByNameParams{
		UserID:     userID,
		DomainName: domainName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}
	return dbDomainToModel(row), nil
}

// GetByUserID retrieves all domains for a user with pagination.
func (r *DomainRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.SendingDomain, error) {
	rows, err := r.queries.GetDomainsByUserIDPaginated(ctx, db.GetDomainsByUserIDPaginatedParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}

	domains := make([]*domain.SendingDomain, len(rows))
	for i, row := range rows {
		domains[i] = dbDomainToModel(row)
	}
	return domains, nil
}

// CountByUserID returns the total number of domains for a user.
func (r *DomainRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	count, err := r.queries.CountDomainsByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count domains: %w", err)
	}
	return int(count), nil
}

// Update updates an existing domain.
func (r *DomainRepository) Update(ctx context.Context, d *domain.SendingDomain) error {
	err := r.queries.UpdateDomain(ctx, db.UpdateDomainParams{
		ID:                 d.ID,
		Status:             string(d.Status),
		VerifiedForSending: d.VerifiedForSending,
		DkimTokens:         d.DkimTokens,
		DkimStatus:         string(d.DkimStatus),
		MailFromDomain:     toPgText(d.MailFromDomain),
		MailFromStatus:     toPgTextFromStatus(d.MailFromStatus),
		LastVerifiedAt:     toPgTimestampFromTimePtr(d.LastCheckedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to update domain: %w", err)
	}
	return nil
}

// Delete removes a domain.
func (r *DomainRepository) Delete(ctx context.Context, id string) error {
	err := r.queries.DeleteDomain(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}
	return nil
}

// dbDomainToModel converts a sqlc Domain to domain.SendingDomain.
func dbDomainToModel(d db.Domain) *domain.SendingDomain {
	result := &domain.SendingDomain{
		ID:                 d.ID,
		UserID:             d.UserID,
		Domain:             d.DomainName,
		Status:             domain.DomainStatus(d.Status),
		VerifiedForSending: d.VerifiedForSending,
		DkimTokens:         d.DkimTokens,
		DkimStatus:         domain.DomainStatus(d.DkimStatus),
		Region:             d.Region,
	}

	if d.MailFromDomain.Valid {
		result.MailFromDomain = d.MailFromDomain.String
	}
	if d.MailFromStatus.Valid {
		result.MailFromStatus = domain.DomainStatus(d.MailFromStatus.String)
	}
	if d.CreatedAt.Valid {
		result.CreatedAt = d.CreatedAt.Time
	}
	if d.UpdatedAt.Valid {
		result.UpdatedAt = d.UpdatedAt.Time
	}
	if d.LastVerifiedAt.Valid {
		result.LastCheckedAt = &d.LastVerifiedAt.Time
	}

	return result
}

// Helper functions for conversions
func toPgTextFromStatus(s domain.DomainStatus) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: string(s), Valid: true}
}

// toPgTimestampFromTimePtr converts a *time.Time to pgtype.Timestamptz.
func toPgTimestampFromTimePtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
