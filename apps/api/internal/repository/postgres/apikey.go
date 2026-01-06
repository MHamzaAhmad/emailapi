package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/domain"
)

// APIKeyRepository implements repository.APIKeyRepository using sqlc-generated queries.
type APIKeyRepositoryImpl struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewAPIKeyRepository creates a new APIKeyRepository.
func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepositoryImpl {
	return &APIKeyRepositoryImpl{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Create stores a new API key using sqlc.
func (r *APIKeyRepositoryImpl) Create(ctx context.Context, apiKey *domain.APIKey) error {
	scopes := make([]string, len(apiKey.Scopes))
	for i, s := range apiKey.Scopes {
		scopes[i] = string(s)
	}

	result, err := r.queries.CreateApiKey(ctx, db.CreateApiKeyParams{
		ID:        apiKey.ID,
		UserID:    apiKey.UserID,
		Name:      apiKey.Name,
		KeyHash:   apiKey.KeyHash,
		KeyPrefix: apiKey.KeyPrefix,
		Scopes:    scopes,
		IsActive:  apiKey.IsActive,
		ExpiresAt: toPgTimestamptzPtr(apiKey.ExpiresAt),
		CreatedAt: toPgTimestampNow(),
		UpdatedAt: toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	apiKey.CreatedAt = result.CreatedAt.Time
	apiKey.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// GetByID retrieves an API key by its ID using sqlc.
func (r *APIKeyRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	row, err := r.queries.GetApiKeyByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	return dbApiKeyToDomain(row), nil
}

// GetByHash retrieves an active API key by its hash using sqlc.
func (r *APIKeyRepositoryImpl) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	row, err := r.queries.GetApiKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key by hash: %w", err)
	}
	return dbApiKeyToDomain(row), nil
}

// GetByPrefix retrieves an API key by its prefix using sqlc.
func (r *APIKeyRepositoryImpl) GetByPrefix(ctx context.Context, keyPrefix string) (*domain.APIKey, error) {
	row, err := r.queries.GetApiKeyByPrefix(ctx, keyPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key by prefix: %w", err)
	}
	return dbApiKeyToDomain(row), nil
}

// ListByUserID retrieves all API keys for a user with pagination using sqlc.
func (r *APIKeyRepositoryImpl) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.APIKey, error) {
	rows, err := r.queries.ListApiKeysByUserIDPaginated(ctx, db.ListApiKeysByUserIDPaginatedParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	apiKeys := make([]*domain.APIKey, len(rows))
	for i, row := range rows {
		apiKeys[i] = dbListApiKeyRowToDomain(row)
	}
	return apiKeys, nil
}

// CountByUserID counts the total number of API keys for a user.
func (r *APIKeyRepositoryImpl) CountByUserID(ctx context.Context, userID string) (int, error) {
	count, err := r.queries.CountApiKeysByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count API keys: %w", err)
	}
	return int(count), nil
}

// Update updates an existing API key using sqlc.
func (r *APIKeyRepositoryImpl) Update(ctx context.Context, apiKey *domain.APIKey) error {
	scopes := make([]string, len(apiKey.Scopes))
	for i, s := range apiKey.Scopes {
		scopes[i] = string(s)
	}

	result, err := r.queries.UpdateApiKey(ctx, db.UpdateApiKeyParams{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Scopes:    scopes,
		IsActive:  apiKey.IsActive,
		ExpiresAt: toPgTimestamptzPtr(apiKey.ExpiresAt),
		UpdatedAt: toPgTimestampNow(),
	})
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	apiKey.UpdatedAt = result.UpdatedAt.Time
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp using sqlc.
func (r *APIKeyRepositoryImpl) UpdateLastUsed(ctx context.Context, id string) error {
	err := r.queries.UpdateApiKeyLastUsed(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}
	return nil
}

// Revoke deactivates an API key using sqlc.
func (r *APIKeyRepositoryImpl) Revoke(ctx context.Context, id string) (*domain.APIKey, error) {
	row, err := r.queries.RevokeApiKey(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke API key: %w", err)
	}
	return dbApiKeyToDomain(row), nil
}

// Delete removes an API key using sqlc.
func (r *APIKeyRepositoryImpl) Delete(ctx context.Context, id string) error {
	err := r.queries.DeleteApiKey(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	return nil
}

// CountActiveByUserID counts active API keys for a user using sqlc.
func (r *APIKeyRepositoryImpl) CountActiveByUserID(ctx context.Context, userID string) (int64, error) {
	count, err := r.queries.CountActiveApiKeysByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count active API keys: %w", err)
	}
	return count, nil
}

// dbApiKeyToDomain converts a sqlc ApiKey to domain.APIKey.
func dbApiKeyToDomain(row db.ApiKey) *domain.APIKey {
	scopes := make([]domain.Scope, len(row.Scopes))
	for i, s := range row.Scopes {
		scopes[i] = domain.Scope(s)
	}

	apiKey := &domain.APIKey{
		ID:        row.ID,
		UserID:    row.UserID,
		Name:      row.Name,
		KeyHash:   row.KeyHash,
		KeyPrefix: row.KeyPrefix,
		Scopes:    scopes,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	if row.LastUsedAt.Valid {
		apiKey.LastUsedAt = &row.LastUsedAt.Time
	}
	if row.ExpiresAt.Valid {
		apiKey.ExpiresAt = &row.ExpiresAt.Time
	}

	return apiKey
}

// dbListApiKeyRowToDomain converts a sqlc ListApiKeysByUserIDPaginatedRow to domain.APIKey.
func dbListApiKeyRowToDomain(row db.ListApiKeysByUserIDPaginatedRow) *domain.APIKey {
	scopes := make([]domain.Scope, len(row.Scopes))
	for i, s := range row.Scopes {
		scopes[i] = domain.Scope(s)
	}

	apiKey := &domain.APIKey{
		ID:        row.ID,
		UserID:    row.UserID,
		Name:      row.Name,
		KeyPrefix: row.KeyPrefix,
		Scopes:    scopes,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	if row.LastUsedAt.Valid {
		apiKey.LastUsedAt = &row.LastUsedAt.Time
	}
	if row.ExpiresAt.Valid {
		apiKey.ExpiresAt = &row.ExpiresAt.Time
	}

	return apiKey
}

// toPgTimestamptzPtr converts a *time.Time to pgtype.Timestamptz.
func toPgTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
