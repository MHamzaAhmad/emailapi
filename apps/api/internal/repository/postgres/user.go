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
type UserRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Create stores a new user using sqlc.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	result, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		Role:       string(user.Role),
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
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return dbUserByIDToDomain(row), nil
}

// GetByEmail retrieves a user by their email using sqlc.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return dbUserByEmailToDomain(row), nil
}

// GetByExternalID retrieves a user by their Clerk external ID.
func (r *UserRepository) GetByExternalID(ctx context.Context, externalID string) (*domain.User, error) {
	row, err := r.queries.GetUserByExternalID(ctx, pgtype.Text{String: externalID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by external_id: %w", err)
	}
	return dbUserByExternalIDToDomain(row), nil
}

// Update updates an existing user using sqlc.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	result, err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:         user.ID,
		Email:      user.Email,
		Name:       user.Name,
		Role:       string(user.Role),
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
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	err := r.queries.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// List retrieves all users with pagination using sqlc.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
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

// Conversion helpers for different SQLC row types

func dbUserByIDToDomain(row db.GetUserByIDRow) *domain.User {
	return &domain.User{
		ID:         row.ID,
		Email:      row.Email,
		Name:       row.Name,
		Role:       domain.UserRole(row.Role),
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
		Role:       domain.UserRole(row.Role),
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
		Role:       domain.UserRole(row.Role),
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
		Role:       domain.UserRole(row.Role),
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
