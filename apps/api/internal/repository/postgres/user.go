package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/emailapi/api/internal/domain"
)

// UserRepository implements repository.UserRepository using PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// Create stores a new user.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, name, role, api_key_hash, api_key_prefix, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Name, user.Role,
		user.APIKey, user.APIKeyPrefix, user.IsActive,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, name, role, api_key_prefix, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role,
		&user.APIKeyPrefix, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by their email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, name, role, api_key_prefix, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role,
		&user.APIKeyPrefix, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetByAPIKey retrieves a user by their API key.
func (r *UserRepository) GetByAPIKey(ctx context.Context, apiKeyHash string) (*domain.User, error) {
	query := `
		SELECT id, email, name, role, api_key_prefix, is_active, created_at, updated_at
		FROM users
		WHERE api_key_hash = $1 AND is_active = true
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, apiKeyHash).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role,
		&user.APIKeyPrefix, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by API key: %w", err)
	}

	return &user, nil
}

// Update updates an existing user.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $2, name = $3, role = $4, is_active = $5, updated_at = $6
		WHERE id = $1
	`

	user.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Email, user.Name, user.Role, user.IsActive, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete removes a user.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// UpdateAPIKey updates a user's API key.
func (r *UserRepository) UpdateAPIKey(ctx context.Context, id string, hashedKey string, prefix string) error {
	query := `UPDATE users SET api_key_hash = $2, api_key_prefix = $3, updated_at = $4 WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, hashedKey, prefix, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	return nil
}
