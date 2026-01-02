package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/domain"
)

// UserRepository implements repository.UserRepository using sqlc-generated queries.
type UserRepositoryImpl struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Create stores a new user using sqlc.
func (r *UserRepositoryImpl) Create(ctx context.Context, user *domain.User) error {
	result, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		Role:       user.Role,
		IsActive:   user.IsActive,
		ExternalID: stringPtrToPgText(user.ExternalID),
		CreatedAt:  toPgTimestampNow(),
		UpdatedAt:  toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.CreatedAt = result.CreatedAt.Time
	user.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// GetByID retrieves a user by their ID using sqlc.
func (r *UserRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return dbUserByIDToDomain(row), nil
}

// GetByEmail retrieves a user by their email using sqlc.
func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return dbUserByEmailToDomain(row), nil
}

// GetByExternalID retrieves a user by their Clerk external ID.
func (r *UserRepositoryImpl) GetByExternalID(ctx context.Context, externalID string) (*domain.User, error) {
	row, err := r.queries.GetUserByExternalID(ctx, pgtype.Text{String: externalID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by external_id: %w", err)
	}
	return dbUserByExternalIDToDomain(row), nil
}

// Update updates an existing user using sqlc.
func (r *UserRepositoryImpl) Update(ctx context.Context, user *domain.User) error {
	result, err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		Role:       user.Role,
		IsActive:   user.IsActive,
		ExternalID: stringPtrToPgText(user.ExternalID),
		UpdatedAt:  toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	user.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// Delete removes a user using sqlc.
func (r *UserRepositoryImpl) Delete(ctx context.Context, id string) error {
	err := r.queries.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// List retrieves all users with pagination using sqlc.
func (r *UserRepositoryImpl) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	rows, err := r.queries.ListUsers(ctx, db.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*domain.User, len(rows))
	for i, row := range rows {
		users[i] = dbListUserToDomain(row)
	}
	return users, nil
}

// Count returns total number of users.
func (r *UserRepositoryImpl) Count(ctx context.Context) (int, error) {
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return int(count), nil
}

// ListWithReputation retrieves users with their suspension/flag status.
func (r *UserRepositoryImpl) ListWithReputation(ctx context.Context, limit, offset int) ([]*domain.AdminUser, error) {
	rows, err := r.queries.ListUsersWithReputation(ctx, db.ListUsersWithReputationParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users with reputation: %w", err)
	}

	users := make([]*domain.AdminUser, len(rows))
	for i, row := range rows {
		users[i] = &domain.AdminUser{
			User: &domain.User{
				ID:         row.ID,
				Email:      row.Email,
				Name:       row.Name,
				Role:       domain.UserRole(row.Role),
				IsActive:   row.IsActive,
				ExternalID: pgTextToStringPtr(row.ExternalID),
				CreatedAt:  row.CreatedAt.Time,
				UpdatedAt:  row.UpdatedAt.Time,
			},
			IsSuspended: row.IsSuspended,
			IsFlagged:   row.IsFlagged,
		}
	}
	return users, nil
}

// GetWithReputation retrieves a user with full reputation data.
func (r *UserRepositoryImpl) GetWithReputation(ctx context.Context, userID string) (*domain.UserWithReputation, error) {
	row, err := r.queries.GetUserWithReputation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with reputation: %w", err)
	}

	result := &domain.UserWithReputation{
		ID:              row.ID,
		Email:           row.Email,
		Name:            row.Name,
		Role:            domain.UserRole(row.Role),
		IsActive:        row.IsActive,
		ExternalID:      pgTextToStringPtr(row.ExternalID),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
		TotalBounces:    int(row.TotalBounces),
		HardBounces:     int(row.HardBounces),
		SoftBounces:     int(row.SoftBounces),
		Complaints:      int(row.Complaints),
		Bounces30d:      int(row.Bounces30d),
		Complaints30d:   int(row.Complaints30d),
		SuspensionScore: numericToFloat64(row.SuspensionScore),
		IsFlagged:       row.IsFlagged,
		IsSuspended:     row.IsSuspended,
	}

	if row.FlaggedAt.Valid {
		result.FlaggedAt = &row.FlaggedAt.Time
	}
	if row.FlaggedReason.Valid {
		result.FlaggedReason = &row.FlaggedReason.String
	}
	if row.SuspendedAt.Valid {
		result.SuspendedAt = &row.SuspendedAt.Time
	}
	if row.SuspendedBy.Valid {
		result.SuspendedBy = &row.SuspendedBy.String
	}
	if row.SuspensionReason.Valid {
		result.SuspensionReason = &row.SuspensionReason.String
	}

	return result, nil
}

// numericToFloat64 converts pgtype.Numeric to float64.
func numericToFloat64(n interface{}) float64 {
	// Handle different possible types from sqlc
	switch v := n.(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	default:
		return 0
	}
}

// Conversion helpers for different SQLC row types

func dbUserByIDToDomain(row db.GetUserByIDRow) *domain.User {
	return &domain.User{
		ID:         row.ID,
		Email:      row.Email,
		Name:       row.Name,
		Role:       row.Role,
		IsActive:   row.IsActive,
		ExternalID: pgTextToStringPtr(row.ExternalID),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func dbUserByEmailToDomain(row db.GetUserByEmailRow) *domain.User {
	return &domain.User{
		ID:         row.ID,
		Email:      row.Email,
		Name:       row.Name,
		Role:       row.Role,
		IsActive:   row.IsActive,
		ExternalID: pgTextToStringPtr(row.ExternalID),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func dbUserByExternalIDToDomain(row db.GetUserByExternalIDRow) *domain.User {
	return &domain.User{
		ID:         row.ID,
		Email:      row.Email,
		Name:       row.Name,
		Role:       row.Role,
		IsActive:   row.IsActive,
		ExternalID: pgTextToStringPtr(row.ExternalID),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func dbListUserToDomain(row db.ListUsersRow) *domain.User {
	return &domain.User{
		ID:         row.ID,
		Email:      row.Email,
		Name:       row.Name,
		Role:       row.Role,
		IsActive:   row.IsActive,
		ExternalID: pgTextToStringPtr(row.ExternalID),
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

func stringPtrToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func pgTextToStringPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
