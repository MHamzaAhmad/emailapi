package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/emailapi/api/internal/domain"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	rediscache "github.com/emailapi/api/internal/repository/redis"
)

// Argon2id parameters for API key hashing
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// APIKeyService handles API key business logic.
type APIKeyService struct {
	store    Store
	cache    rediscache.APIKeyCacheInterface
	activity chrepo.ActivityRepositoryInterface
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(store Store, cache rediscache.APIKeyCacheInterface, activity chrepo.ActivityRepositoryInterface) *APIKeyService {
	return &APIKeyService{store: store, cache: cache, activity: activity}
}

// Create creates a new API key for a user.
// Returns the API key and the raw key (only shown once).
func (s *APIKeyService) Create(ctx context.Context, userID string, req *domain.CreateAPIKeyRequest) (*domain.APIKey, string, error) {
	// Validate user exists
	_, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("user not found: %w", err)
	}

	// Set default environment if not specified
	env := req.Environment
	if env == "" {
		env = domain.EnvLive
	}

	// Set default scopes if not specified
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = domain.DefaultScopes()
	}

	// Validate scopes
	if !domain.ValidateScopes(scopes) {
		return nil, "", fmt.Errorf("invalid scopes provided")
	}

	// Generate API key with ep_{env}_ prefix format
	rawKey, err := s.generateRawKey(env)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the key for storage
	hashedKey, err := s.hashKey(rawKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}

	// Generate display prefix (first 16 chars + ...)
	keyPrefix := rawKey[:16] + "..."

	// Generate unique ID (non-UUID, alphanumeric)
	id, err := s.generateKeyID()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key ID: %w", err)
	}

	apiKey := &domain.APIKey{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		KeyHash:     hashedKey,
		KeyPrefix:   keyPrefix,
		Scopes:      scopes,
		Environment: env,
		IsActive:    true,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := s.store.APIKeys().Create(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	// Invalidate user's API keys list cache
	if s.cache != nil {
		_ = s.cache.InvalidateByUserID(ctx, userID)
	}

	// Log activity
	if s.activity != nil {
		_ = s.activity.LogAPIKey(ctx, userID, apiKey.ID, "create", "success", fmt.Sprintf("API Key %s created", apiKey.Name))
	}

	return apiKey, rawKey, nil
}

// GetByID retrieves an API key by ID.
func (s *APIKeyService) GetByID(ctx context.Context, userID, keyID string) (*domain.APIKey, error) {
	// Try cache first
	if s.cache != nil {
		if cached, _ := s.cache.GetByID(ctx, keyID); cached != nil {
			if cached.UserID == userID {
				return cached, nil
			}
		}
	}

	apiKey, err := s.store.APIKeys().GetByID(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Authorization check
	if apiKey.UserID != userID {
		return nil, fmt.Errorf("API key not found")
	}

	// Cache the result
	if s.cache != nil {
		_ = s.cache.SetByID(ctx, apiKey)
	}

	return apiKey, nil
}

// List retrieves all API keys for a user with pagination.
// Uses cache for unpaginated requests (page <= 1, pageSize >= 100) for performance.
func (s *APIKeyService) List(ctx context.Context, userID string, page, pageSize int) ([]*domain.APIKey, int, error) {
	// Default pagination if not specified
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 100
	}

	// Use cache for "list all" requests (first page with large page size)
	// This optimizes the common case of loading API keys in UI
	useCache := page == 1 && pageSize >= 100

	if useCache && s.cache != nil {
		if cached, _ := s.cache.GetByUserID(ctx, userID); cached != nil {
			// Return cached results (limited to pageSize for consistency)
			limit := len(cached)
			if pageSize < limit {
				limit = pageSize
			}

			result := make([]*domain.APIKey, limit)
			for i := 0; i < limit; i++ {
				result[i] = cached[i]
			}
			return result, len(cached), nil
		}
	}

	offset := (page - 1) * pageSize

	keys, err := s.store.APIKeys().ListByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	// Get total count for pagination
	total, err := s.store.APIKeys().CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count API keys: %w", err)
	}

	// Cache the result for first page
	if useCache && s.cache != nil && page == 1 {
		_ = s.cache.SetByUserID(ctx, userID, keys)
	}

	return keys, total, nil
}

// Update updates an API key.
func (s *APIKeyService) Update(ctx context.Context, userID, keyID string, req *domain.UpdateAPIKeyRequest) (*domain.APIKey, error) {
	apiKey, err := s.GetByID(ctx, userID, keyID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if len(req.Scopes) > 0 {
		if !domain.ValidateScopes(req.Scopes) {
			return nil, fmt.Errorf("invalid scopes provided")
		}
		apiKey.Scopes = req.Scopes
	}
	if req.IsActive != nil {
		apiKey.IsActive = *req.IsActive
	}
	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}

	if err := s.store.APIKeys().Update(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateAll(ctx, apiKey.ID, apiKey.KeyPrefix, userID)
	}

	return apiKey, nil
}

// Delete deletes an API key.
func (s *APIKeyService) Delete(ctx context.Context, userID, keyID string) error {
	// Verify ownership and get key for cache invalidation
	apiKey, err := s.GetByID(ctx, userID, keyID)
	if err != nil {
		return err
	}

	if err := s.store.APIKeys().Delete(ctx, keyID); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateAll(ctx, apiKey.ID, apiKey.KeyPrefix, userID)
	}

	// Log activity
	if s.activity != nil {
		_ = s.activity.LogAPIKey(ctx, userID, apiKey.ID, "delete", "success", fmt.Sprintf("API Key %s deleted", apiKey.Name))
	}

	return nil
}

// Revoke revokes an API key (soft delete).
func (s *APIKeyService) Revoke(ctx context.Context, userID, keyID string) (*domain.APIKey, error) {
	// Verify ownership and get key for cache invalidation
	apiKey, err := s.GetByID(ctx, userID, keyID)
	if err != nil {
		return nil, err
	}

	result, err := s.store.APIKeys().Revoke(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateAll(ctx, apiKey.ID, apiKey.KeyPrefix, userID)
	}

	// Log activity
	if s.activity != nil {
		_ = s.activity.LogAPIKey(ctx, userID, apiKey.ID, "revoke", "success", fmt.Sprintf("API Key %s revoked", apiKey.Name))
	}

	return result, nil
}

// ValidateAndGetUser validates an API key and returns the associated user.
// This is used by authentication middleware.
func (s *APIKeyService) ValidateAndGetUser(ctx context.Context, rawKey string) (*domain.User, *domain.APIKey, error) {
	// Extract environment from key prefix
	if !strings.HasPrefix(rawKey, "ep_") {
		return nil, nil, fmt.Errorf("invalid API key format")
	}

	// Hash the key for lookup
	// First we need to find the key by iterating - this is inefficient
	// In production, you'd use a key prefix lookup first
	// For now, we'll use a different approach - store the salt with the hash

	// Actually, since we're using Argon2id with unique salts per key,
	// we need a different approach. We'll store the hash in a way that
	// allows verification. Standard approach: extract prefix, find candidate keys,
	// then verify.

	// Extract the prefix for lookup (ep_live_ or ep_dev_ plus first few chars)
	if len(rawKey) < 16 {
		return nil, nil, fmt.Errorf("invalid API key format")
	}
	prefix := rawKey[:16] + "..."

	apiKey, err := s.store.APIKeys().GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid API key")
	}

	// Verify the key matches the stored hash
	if !s.verifyKey(rawKey, apiKey.KeyHash) {
		return nil, nil, fmt.Errorf("invalid API key")
	}

	// Check if key is active
	if !apiKey.IsActive {
		return nil, nil, fmt.Errorf("API key is inactive")
	}

	// Check if key is expired
	if apiKey.IsExpired() {
		return nil, nil, fmt.Errorf("API key has expired")
	}

	// Update last used timestamp
	_ = s.store.APIKeys().UpdateLastUsed(ctx, apiKey.ID)

	// Get the user
	user, err := s.store.Users().GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, nil, fmt.Errorf("user is inactive")
	}

	return user, apiKey, nil
}

// generateRawKey generates a raw API key with the ep_{env}_ prefix.
// Format: ep_live_<32 chars base62> or ep_dev_<32 chars base62>
func (s *APIKeyService) generateRawKey(env domain.Environment) (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base62-like string (alphanumeric)
	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	// Remove non-alphanumeric chars and limit to 32 chars
	alphanumeric := strings.ReplaceAll(encoded, "-", "")
	alphanumeric = strings.ReplaceAll(alphanumeric, "_", "")
	if len(alphanumeric) > 32 {
		alphanumeric = alphanumeric[:32]
	}

	return fmt.Sprintf("ep_%s_%s", env, alphanumeric), nil
}

// generateKeyID generates a unique alphanumeric key ID.
// Format: ak_<12 chars alphanumeric>
func (s *APIKeyService) generateKeyID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	alphanumeric := strings.ReplaceAll(encoded, "-", "")
	alphanumeric = strings.ReplaceAll(alphanumeric, "_", "")
	if len(alphanumeric) > 16 {
		alphanumeric = alphanumeric[:16]
	}

	return "ak_" + alphanumeric, nil
}

// hashKey creates an Argon2id hash of the API key.
func (s *APIKeyService) hashKey(key string) (string, error) {
	// Generate salt
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash with Argon2id
	hash := argon2.IDKey([]byte(key), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encode as salt$hash (both base64)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	return saltB64 + "$" + hashB64, nil
}

// verifyKey verifies an API key against its stored hash.
func (s *APIKeyService) verifyKey(key, storedHash string) bool {
	parts := strings.Split(storedHash, "$")
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(key), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Constant-time comparison
	return subtle.ConstantTimeCompare(computedHash, expectedHash) == 1
}

// GetScopes returns the scopes for an API key.
func (s *APIKeyService) GetScopes(apiKey *domain.APIKey) []domain.Scope {
	return apiKey.Scopes
}

// HasScope checks if an API key has a specific scope.
func (s *APIKeyService) HasScope(apiKey *domain.APIKey, scope domain.Scope) bool {
	return apiKey.HasScope(scope)
}

// CountActiveByUserID returns the number of active API keys for a user.
func (s *APIKeyService) CountActiveByUserID(ctx context.Context, userID string) (int64, error) {
	return s.store.APIKeys().CountActiveByUserID(ctx, userID)
}
