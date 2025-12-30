package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/domain"
)

// UnsubscribeRepository implements repository.UnsubscribeRepository using PostgreSQL.
type UnsubscribeRepository struct {
	queries *db.Queries
}

// NewUnsubscribeRepository creates a new UnsubscribeRepository.
func NewUnsubscribeRepository(pool *pgxpool.Pool) *UnsubscribeRepository {
	return &UnsubscribeRepository{
		queries: db.New(pool),
	}
}

// Add adds an email to the unsubscribe list for a user.
func (r *UnsubscribeRepository) Add(ctx context.Context, entry *domain.UnsubscribeEntry) error {
	params := db.InsertUnsubscribeParams{
		ID:        entry.ID,
		UserID:    entry.UserID,
		EmailHash: entry.EmailHash,
		Source:    string(entry.Source),
	}

	if entry.SourceEmailID != "" {
		params.SourceEmailID = pgtype.Text{String: entry.SourceEmailID, Valid: true}
	}

	if err := r.queries.InsertUnsubscribe(ctx, params); err != nil {
		return fmt.Errorf("failed to insert unsubscribe: %w", err)
	}

	return nil
}

// GetByUserAndHash retrieves an unsubscribe entry by user and email hash.
func (r *UnsubscribeRepository) GetByUserAndHash(ctx context.Context, userID, emailHash string) (*domain.UnsubscribeEntry, error) {
	row, err := r.queries.GetUnsubscribeByUserAndHash(ctx, db.GetUnsubscribeByUserAndHashParams{
		UserID:    userID,
		EmailHash: emailHash,
	})
	if err != nil {
		return nil, err
	}

	return &domain.UnsubscribeEntry{
		ID:            row.ID,
		UserID:        row.UserID,
		EmailHash:     row.EmailHash,
		SourceEmailID: row.SourceEmailID.String,
		Source:        domain.UnsubscribeSource(row.Source),
		CreatedAt:     row.CreatedAt.Time,
	}, nil
}

// CheckBatch checks multiple email hashes for a user, returns unsubscribed hashes.
func (r *UnsubscribeRepository) CheckBatch(ctx context.Context, userID string, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	return r.queries.CheckUnsubscribeBatch(ctx, db.CheckUnsubscribeBatchParams{
		UserID:  userID,
		Column2: hashes,
	})
}

// Delete removes an email from the unsubscribe list.
func (r *UnsubscribeRepository) Delete(ctx context.Context, userID, emailHash string) error {
	return r.queries.DeleteUnsubscribe(ctx, db.DeleteUnsubscribeParams{
		UserID:    userID,
		EmailHash: emailHash,
	})
}

// ListByUserID retrieves unsubscribes for a user with pagination.
func (r *UnsubscribeRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.UnsubscribeEntry, error) {
	rows, err := r.queries.ListUnsubscribesByUser(ctx, db.ListUnsubscribesByUserParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list unsubscribes: %w", err)
	}

	entries := make([]*domain.UnsubscribeEntry, len(rows))
	for i, row := range rows {
		entries[i] = &domain.UnsubscribeEntry{
			ID:            row.ID,
			EmailHash:     row.EmailHash,
			SourceEmailID: row.SourceEmailID.String,
			Source:        domain.UnsubscribeSource(row.Source),
			CreatedAt:     row.CreatedAt.Time,
		}
	}

	return entries, nil
}

// CountByUserID returns total unsubscribes for a user.
func (r *UnsubscribeRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return r.queries.CountUnsubscribesByUser(ctx, userID)
}

// ListAll returns all unsubscribes (for cache sync).
func (r *UnsubscribeRepository) ListAll(ctx context.Context) ([]struct{ UserID, EmailHash string }, error) {
	rows, err := r.queries.ListAllUnsubscribes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all unsubscribes: %w", err)
	}

	result := make([]struct{ UserID, EmailHash string }, len(rows))
	for i, row := range rows {
		result[i] = struct{ UserID, EmailHash string }{
			UserID:    row.UserID,
			EmailHash: row.EmailHash,
		}
	}

	return result, nil
}
