package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emailapi/api/internal/domain"
)

// DomainRepository implements repository.DomainRepository using PostgreSQL.
type DomainRepository struct {
	pool *pgxpool.Pool
}

// NewDomainRepository creates a new DomainRepository.
func NewDomainRepository(pool *pgxpool.Pool) *DomainRepository {
	return &DomainRepository{pool: pool}
}

// Create stores a new sending domain.
func (r *DomainRepository) Create(ctx context.Context, d *domain.SendingDomain) error {
	query := `
		INSERT INTO domains (
			id, user_id, domain_name, status, verified_for_sending,
			dkim_tokens, dkim_status, mail_from_domain, mail_from_status,
			region, created_at, updated_at, last_verified_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.pool.Exec(ctx, query,
		d.ID,
		d.UserID,
		d.Domain,
		string(d.Status),
		d.VerifiedForSending,
		d.DkimTokens,
		string(d.DkimStatus),
		toPgText(d.MailFromDomain),
		toPgTextFromStatus(d.MailFromStatus),
		d.Region,
		toPgTimestampFromTime(d.CreatedAt),
		toPgTimestampFromTime(d.UpdatedAt),
		toPgTimestampFromTimePtr(d.LastVerifiedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to create domain: %w", err)
	}

	return nil
}

// GetByID retrieves a domain by its ID.
func (r *DomainRepository) GetByID(ctx context.Context, id string) (*domain.SendingDomain, error) {
	query := `
		SELECT id, user_id, domain_name, status, verified_for_sending,
			   dkim_tokens, dkim_status, mail_from_domain, mail_from_status,
			   region, created_at, updated_at, last_verified_at
		FROM domains WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	return r.scanDomain(row)
}

// GetByDomainName retrieves a domain by its name for a specific user.
func (r *DomainRepository) GetByDomainName(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error) {
	query := `
		SELECT id, user_id, domain_name, status, verified_for_sending,
			   dkim_tokens, dkim_status, mail_from_domain, mail_from_status,
			   region, created_at, updated_at, last_verified_at
		FROM domains WHERE user_id = $1 AND domain_name = $2
	`

	row := r.pool.QueryRow(ctx, query, userID, domainName)
	return r.scanDomain(row)
}

// GetByUserID retrieves all domains for a user.
func (r *DomainRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.SendingDomain, error) {
	query := `
		SELECT id, user_id, domain_name, status, verified_for_sending,
			   dkim_tokens, dkim_status, mail_from_domain, mail_from_status,
			   region, created_at, updated_at, last_verified_at
		FROM domains WHERE user_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}
	defer rows.Close()

	var domains []*domain.SendingDomain
	for rows.Next() {
		d, err := r.scanDomainRow(rows)
		if err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}

	return domains, nil
}

// Update updates an existing domain.
func (r *DomainRepository) Update(ctx context.Context, d *domain.SendingDomain) error {
	query := `
		UPDATE domains SET
			status = $2,
			verified_for_sending = $3,
			dkim_tokens = $4,
			dkim_status = $5,
			mail_from_domain = $6,
			mail_from_status = $7,
			updated_at = NOW(),
			last_verified_at = $8
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		d.ID,
		string(d.Status),
		d.VerifiedForSending,
		d.DkimTokens,
		string(d.DkimStatus),
		toPgText(d.MailFromDomain),
		toPgTextFromStatus(d.MailFromStatus),
		toPgTimestampFromTimePtr(d.LastVerifiedAt),
	)
	if err != nil {
		return fmt.Errorf("failed to update domain: %w", err)
	}

	return nil
}

// Delete removes a domain.
func (r *DomainRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM domains WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	return nil
}

// scanDomain scans a single row into a SendingDomain.
func (r *DomainRepository) scanDomain(row pgx.Row) (*domain.SendingDomain, error) {
	var d domain.SendingDomain
	var status, dkimStatus string
	var mailFromDomain, mailFromStatus pgtype.Text
	var createdAt, updatedAt, lastVerifiedAt pgtype.Timestamptz

	err := row.Scan(
		&d.ID,
		&d.UserID,
		&d.Domain,
		&status,
		&d.VerifiedForSending,
		&d.DkimTokens,
		&dkimStatus,
		&mailFromDomain,
		&mailFromStatus,
		&d.Region,
		&createdAt,
		&updatedAt,
		&lastVerifiedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan domain: %w", err)
	}

	d.Status = domain.DomainStatus(status)
	d.DkimStatus = domain.DomainStatus(dkimStatus)
	if mailFromDomain.Valid {
		d.MailFromDomain = mailFromDomain.String
	}
	if mailFromStatus.Valid {
		d.MailFromStatus = domain.DomainStatus(mailFromStatus.String)
	}
	if createdAt.Valid {
		d.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		d.UpdatedAt = updatedAt.Time
	}
	if lastVerifiedAt.Valid {
		d.LastVerifiedAt = &lastVerifiedAt.Time
	}

	return &d, nil
}

// scanDomainRow scans a row from Rows into a SendingDomain.
func (r *DomainRepository) scanDomainRow(rows pgx.Rows) (*domain.SendingDomain, error) {
	var d domain.SendingDomain
	var status, dkimStatus string
	var mailFromDomain, mailFromStatus pgtype.Text
	var createdAt, updatedAt, lastVerifiedAt pgtype.Timestamptz

	err := rows.Scan(
		&d.ID,
		&d.UserID,
		&d.Domain,
		&status,
		&d.VerifiedForSending,
		&d.DkimTokens,
		&dkimStatus,
		&mailFromDomain,
		&mailFromStatus,
		&d.Region,
		&createdAt,
		&updatedAt,
		&lastVerifiedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan domain: %w", err)
	}

	d.Status = domain.DomainStatus(status)
	d.DkimStatus = domain.DomainStatus(dkimStatus)
	if mailFromDomain.Valid {
		d.MailFromDomain = mailFromDomain.String
	}
	if mailFromStatus.Valid {
		d.MailFromStatus = domain.DomainStatus(mailFromStatus.String)
	}
	if createdAt.Valid {
		d.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		d.UpdatedAt = updatedAt.Time
	}
	if lastVerifiedAt.Valid {
		d.LastVerifiedAt = &lastVerifiedAt.Time
	}

	return &d, nil
}

// Helper functions for domain conversions
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
