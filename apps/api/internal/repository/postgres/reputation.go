package postgres

import (
	"context"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/domain"
)

// ReputationRepository implements repository.ReputationRepository using sqlc-generated queries.
type ReputationRepositoryImpl struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewReputationRepository creates a new ReputationRepository.
func NewReputationRepository(pool *pgxpool.Pool) *ReputationRepositoryImpl {
	return &ReputationRepositoryImpl{
		pool:    pool,
		queries: db.New(pool),
	}
}

// EnsureExists creates a reputation record if it doesn't exist.
func (r *ReputationRepositoryImpl) EnsureExists(ctx context.Context, userID string) error {
	return r.queries.EnsureUserReputation(ctx, userID)
}

// Get retrieves reputation stats for a user.
func (r *ReputationRepositoryImpl) Get(ctx context.Context, userID string) (*domain.UserReputation, error) {
	row, err := r.queries.GetUserReputation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user reputation: %w", err)
	}
	return dbReputationToDomain(row), nil
}

// InsertIncident records a new reputation incident.
func (r *ReputationRepositoryImpl) InsertIncident(ctx context.Context, incident *domain.ReputationIncident) error {
	params := db.InsertReputationIncidentParams{
		ID:                 incident.ID,
		UserID:             incident.UserID,
		IncidentType:       incident.IncidentType,
		MessageID:          incident.MessageID,
		RecipientEmailHash: incident.RecipientEmailHash,
	}

	if incident.BounceType != "" {
		params.BounceType = pgtype.Text{String: incident.BounceType, Valid: true}
	}
	if incident.BounceSubtype != "" {
		params.BounceSubtype = pgtype.Text{String: incident.BounceSubtype, Valid: true}
	}
	if incident.ComplaintFeedbackType != "" {
		params.ComplaintFeedbackType = pgtype.Text{String: incident.ComplaintFeedbackType, Valid: true}
	}
	if incident.DiagnosticCode != "" {
		params.DiagnosticCode = pgtype.Text{String: incident.DiagnosticCode, Valid: true}
	}

	return r.queries.InsertReputationIncident(ctx, params)
}

// CountIncidents returns aggregated incident counts for a user.
func (r *ReputationRepositoryImpl) CountIncidents(ctx context.Context, userID string) (*domain.IncidentStats, error) {
	row, err := r.queries.CountIncidentsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count incidents: %w", err)
	}
	return &domain.IncidentStats{
		HardBounces:   int(row.HardBounces),
		SoftBounces:   int(row.SoftBounces),
		Complaints:    int(row.Complaints),
		Bounces30d:    int(row.Bounces30d),
		Complaints30d: int(row.Complaints30d),
	}, nil
}

// UpdateStats updates the reputation statistics.
func (r *ReputationRepositoryImpl) UpdateStats(ctx context.Context, userID string, stats *domain.UserReputation) error {
	params := db.UpdateUserReputationStatsParams{
		UserID:          userID,
		TotalBounces:    int32(stats.TotalBounces),
		HardBounces:     int32(stats.HardBounces),
		SoftBounces:     int32(stats.SoftBounces),
		Complaints:      int32(stats.Complaints),
		Bounces30d:      int32(stats.Bounces30d),
		Complaints30d:   int32(stats.Complaints30d),
		SuspensionScore: floatToNumeric(stats.SuspensionScore),
		IsFlagged:       stats.IsFlagged,
	}

	if stats.FlaggedReason != "" {
		params.FlaggedReason = pgtype.Text{String: stats.FlaggedReason, Valid: true}
	}

	return r.queries.UpdateUserReputationStats(ctx, params)
}

// Suspend marks a user as suspended.
func (r *ReputationRepositoryImpl) Suspend(ctx context.Context, userID, suspendedBy, reason string) error {
	return r.queries.SuspendUserReputation(ctx, db.SuspendUserReputationParams{
		UserID:           userID,
		SuspendedBy:      pgtype.Text{String: suspendedBy, Valid: true},
		SuspensionReason: pgtype.Text{String: reason, Valid: true},
	})
}

// Unsuspend removes suspension from a user.
func (r *ReputationRepositoryImpl) Unsuspend(ctx context.Context, userID string) error {
	return r.queries.UnsuspendUserReputation(ctx, userID)
}

// ListFlagged lists flagged users with pagination.
func (r *ReputationRepositoryImpl) ListFlagged(ctx context.Context, limit, offset int) ([]*domain.UserReputation, int, error) {
	rows, err := r.queries.ListFlaggedUserReputations(ctx, db.ListFlaggedUserReputationsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list flagged users: %w", err)
	}

	total, err := r.queries.CountFlaggedUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count flagged users: %w", err)
	}

	result := make([]*domain.UserReputation, len(rows))
	for i, row := range rows {
		result[i] = dbFlaggedReputationToDomain(row)
	}

	return result, int(total), nil
}

// ListIncidents lists incidents for a user with pagination.
func (r *ReputationRepositoryImpl) ListIncidents(ctx context.Context, userID string, limit, offset int) ([]*domain.ReputationIncident, error) {
	rows, err := r.queries.ListReputationIncidents(ctx, db.ListReputationIncidentsParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}

	result := make([]*domain.ReputationIncident, len(rows))
	for i, row := range rows {
		result[i] = dbIncidentToDomain(row)
	}

	return result, nil
}

// Conversion helpers

func dbReputationToDomain(row db.UserReputation) *domain.UserReputation {
	rep := &domain.UserReputation{
		ID:              row.ID,
		UserID:          row.UserID,
		TotalBounces:    int(row.TotalBounces),
		HardBounces:     int(row.HardBounces),
		SoftBounces:     int(row.SoftBounces),
		Complaints:      int(row.Complaints),
		Bounces30d:      int(row.Bounces30d),
		Complaints30d:   int(row.Complaints30d),
		SuspensionScore: numericToFloat(row.SuspensionScore),
		IsFlagged:       row.IsFlagged,
		IsSuspended:     row.IsSuspended,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}

	if row.FlaggedAt.Valid {
		rep.FlaggedAt = &row.FlaggedAt.Time
	}
	if row.FlaggedReason.Valid {
		rep.FlaggedReason = row.FlaggedReason.String
	}
	if row.SuspendedAt.Valid {
		rep.SuspendedAt = &row.SuspendedAt.Time
	}
	if row.SuspendedBy.Valid {
		rep.SuspendedBy = row.SuspendedBy.String
	}
	if row.SuspensionReason.Valid {
		rep.SuspensionReason = row.SuspensionReason.String
	}

	return rep
}

func dbFlaggedReputationToDomain(row db.ListFlaggedUserReputationsRow) *domain.UserReputation {
	rep := &domain.UserReputation{
		ID:              row.ID,
		UserID:          row.UserID,
		TotalBounces:    int(row.TotalBounces),
		HardBounces:     int(row.HardBounces),
		SoftBounces:     int(row.SoftBounces),
		Complaints:      int(row.Complaints),
		Bounces30d:      int(row.Bounces30d),
		Complaints30d:   int(row.Complaints30d),
		SuspensionScore: numericToFloat(row.SuspensionScore),
		IsFlagged:       row.IsFlagged,
		IsSuspended:     row.IsSuspended,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
		UserEmail:       row.UserEmail,
		UserName:        row.UserName,
	}

	if row.FlaggedAt.Valid {
		rep.FlaggedAt = &row.FlaggedAt.Time
	}
	if row.FlaggedReason.Valid {
		rep.FlaggedReason = row.FlaggedReason.String
	}
	if row.SuspendedAt.Valid {
		rep.SuspendedAt = &row.SuspendedAt.Time
	}
	if row.SuspendedBy.Valid {
		rep.SuspendedBy = row.SuspendedBy.String
	}
	if row.SuspensionReason.Valid {
		rep.SuspensionReason = row.SuspensionReason.String
	}

	return rep
}

func dbIncidentToDomain(row db.ReputationIncident) *domain.ReputationIncident {
	incident := &domain.ReputationIncident{
		ID:                 row.ID,
		UserID:             row.UserID,
		IncidentType:       row.IncidentType,
		MessageID:          row.MessageID,
		RecipientEmailHash: row.RecipientEmailHash,
		CreatedAt:          row.CreatedAt.Time,
	}

	if row.BounceType.Valid {
		incident.BounceType = row.BounceType.String
	}
	if row.BounceSubtype.Valid {
		incident.BounceSubtype = row.BounceSubtype.String
	}
	if row.ComplaintFeedbackType.Valid {
		incident.ComplaintFeedbackType = row.ComplaintFeedbackType.String
	}
	if row.DiagnosticCode.Valid {
		incident.DiagnosticCode = row.DiagnosticCode.String
	}

	return incident
}

func floatToNumeric(f float64) pgtype.Numeric {
	// Convert float64 to pgtype.Numeric with 2 decimal places
	// Multiply by 100, convert to int, then set exp to -2
	intVal := int64(f * 100)
	return pgtype.Numeric{
		Int:   big.NewInt(intVal),
		Exp:   -2,
		Valid: true,
	}
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}

	// Convert the big.Int to float64 and apply the exponent
	f := new(big.Float).SetInt(n.Int)

	// Apply the exponent
	if n.Exp < 0 {
		divisor := new(big.Float).SetFloat64(1)
		for i := int32(0); i > n.Exp; i-- {
			divisor.Mul(divisor, big.NewFloat(10))
		}
		f.Quo(f, divisor)
	} else if n.Exp > 0 {
		for i := int32(0); i < n.Exp; i++ {
			f.Mul(f, big.NewFloat(10))
		}
	}

	result, _ := f.Float64()
	return result
}
