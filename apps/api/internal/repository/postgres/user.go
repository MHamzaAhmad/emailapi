package postgres

import (
	"context"
	"fmt"

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
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		Role:         string(user.Role),
		ApiKeyHash:   toPgTextPtr(user.APIKey),
		ApiKeyPrefix: toPgTextPtr(user.APIKeyPrefix),
		IsActive:     user.IsActive,
		CreatedAt:    toPgTimestampNow(),
		UpdatedAt:    toPgTimestampNow(),
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
	return toDomainUserFromRow(row), nil
}

// GetByEmail retrieves a user by their email using sqlc.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return toDomainUserFromRow(row), nil
}

// GetByAPIKey retrieves a user by their API key hash using sqlc.
func (r *UserRepository) GetByAPIKey(ctx context.Context, apiKeyHash string) (*domain.User, error) {
	row, err := r.queries.GetUserByAPIKey(ctx, apiKeyHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by API key: %w", err)
	}
	return toDomainUserFromRow(row), nil
}

// Update updates an existing user using sqlc.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	result, err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		IsActive:  user.IsActive,
		UpdatedAt: toPgTimestampNow(),
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

// UpdateAPIKey updates a user's API key using sqlc.
func (r *UserRepository) UpdateAPIKey(ctx context.Context, id string, hashedKey string, prefix string) error {
	err := r.queries.UpdateUserAPIKey(ctx, db.UpdateUserAPIKeyParams{
		ID:           id,
		ApiKeyHash:   toPgText(hashedKey),
		ApiKeyPrefix: toPgText(prefix),
	})
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}
	return nil
}

// toDomainUserFromRow converts a sqlc GetUserByID/Email/APIKey Row to domain.User.
func toDomainUserFromRow(row db.GetUserByIDRow) *domain.User {
	return &domain.User{
		ID:           row.ID,
		Email:        row.Email,
		Name:         row.Name,
		Role:         domain.UserRole(row.Role),
		APIKeyPrefix: fromPgTextPtr(row.ApiKeyPrefix),
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
