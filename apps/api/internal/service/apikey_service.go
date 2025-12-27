package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"strings"

	"github.com/emailapi/api/internal/domain"
	rediscache "github.com/emailapi/api/internal/repository/redis"
	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
)

const (
	apiKeySecretLength   = 32 // 32 characters for the secret part
	apiKeyChecksumLength = 4  // 4 characters for CRC32 checksum
)

// APIKeyService handles API key business logic.
type APIKeyService struct {
	store      Store
	cache      rediscache.APIKeyCacheInterface
	activity   tbrepo.ActivityRepositoryInterface
	hmacSecret []byte
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(store Store, cache rediscache.APIKeyCacheInterface, activity tbrepo.ActivityRepositoryInterface, hmacSecret string) *APIKeyService {
	return &APIKeyService{
		store:      store,
		cache:      cache,
		activity:   activity,
		hmacSecret: []byte(hmacSecret),
	}
}

// Create creates a new API key for a user.
// Returns the API key and the raw key (only shown once).
func (s *APIKeyService) Create(ctx context.Context, userID string, req *domain.CreateAPIKeyRequest) (*domain.APIKey, string, error) {
	// Validate user exists
	_, err := s.store.Users().GetByID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("user not found: %w", err)
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

	// Generate API key with ep_ prefix format
	rawKey, err := s.generateRawKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the key for storage
	hashedKey := s.hashKey(rawKey)

	// Generate display prefix (first 12 chars + ...)
	keyPrefix := rawKey[:12] + "..."

	// Generate unique ID (non-UUID, alphanumeric)
	id, err := s.generateKeyID()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key ID: %w", err)
	}

	apiKey := &domain.APIKey{
		ID:        id,
		UserID:    userID,
		Name:      req.Name,
		KeyHash:   hashedKey,
		KeyPrefix: keyPrefix,
		Scopes:    scopes,
		IsActive:  true,
		ExpiresAt: req.ExpiresAt,
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
// Fast-path optimization: Checksum -> Redis Cache -> DB fallback.
func (s *APIKeyService) ValidateAndGetUser(ctx context.Context, rawKey string) (*domain.User, *domain.APIKey, error) {
	//  Step 1: Validate checksum (instant rejection for invalid keys)
	if !s.validateChecksum(rawKey) {
		return nil, nil, fmt.Errorf("invalid API key format or checksum")
	}

	// Step 2: Hash the key for lookups
	keyHash := s.hashKey(rawKey)

	// Step 3: Try Redis cache first (fast path)
	if s.cache != nil {
		if cachedKey, _ := s.cache.GetByKeyHash(ctx, keyHash); cachedKey != nil {
			// Verify key is still active and not expired
			if !cachedKey.IsActive {
				return nil, nil, fmt.Errorf("API key is inactive")
			}
			if cachedKey.IsExpired() {
				return nil, nil, fmt.Errorf("API key has expired")
			}

			// Get user (this should also be cached)
			user, err := s.store.Users().GetByID(ctx, cachedKey.UserID)
			if err != nil {
				return nil, nil, fmt.Errorf("user not found")
			}

			if !user.IsActive {
				return nil, nil, fmt.Errorf("user is inactive")
			}

			// Async update last used (fire and forget)
			go func() {
				_ = s.store.APIKeys().UpdateLastUsed(context.Background(), cachedKey.ID)
			}()

			return user, cachedKey, nil
		}
	}

	// Step 4: Cache miss - look up in DB by hash
	apiKey, err := s.store.APIKeys().GetByHash(ctx, keyHash)
	if err != nil {
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

	// Cache the validated key for next time
	if s.cache != nil {
		_ = s.cache.SetByKeyHash(ctx, keyHash, apiKey)
	}

	// Async update last used (fire and forget)
	go func() {
		_ = s.store.APIKeys().UpdateLastUsed(context.Background(), apiKey.ID)
	}()

	// Get  the user
	user, err := s.store.Users().GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, nil, fmt.Errorf("user is inactive")
	}

	return user, apiKey, nil
}

// generateRawKey generates a raw API key with checksum.
// Format: sea_live_<32 chars>_<4 char checksum>
func (s *APIKeyService) generateRawKey() (string, error) {
	// Generate secret part
	secret, err := s.generateSecret(apiKeySecretLength)
	if err != nil {
		return "", err
	}

	// Generate checksum
	checksum := s.generateChecksum(secret)

	// Format: sea_live_<secret>_<checksum>
	return fmt.Sprintf("sea_live_%s_%s", secret, checksum), nil
}

// generateSecret generates a cryptographically random secret string.
func (s *APIKeyService) generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base64 URL encoding (alphanumeric + - and _)
	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	// Remove - and _ to make it cleaner (optional)
	alphanumeric := strings.ReplaceAll(encoded, "-", "")
	alphanumeric = strings.ReplaceAll(alphanumeric, "_", "")

	// Truncate to desired length
	if len(alphanumeric) > length {
		alphanumeric = alphanumeric[:length]
	}

	return alphanumeric, nil
}

// generateChecksum generates a CRC32 checksum for the secret.
func (s *APIKeyService) generateChecksum(secret string) string {
	crc := crc32.ChecksumIEEE([]byte(secret))
	// Return last 4 hex digits
	return fmt.Sprintf("%04x", crc&0xFFFF)
}

// validateChecksum validates that the key's checksum matches its secret.
func (s *APIKeyService) validateChecksum(key string) bool {
	// Parse key format: sea_live_<secret>_<checksum>
	parts := strings.Split(key, "_")
	if len(parts) != 4 {
		return false // Invalid format
	}

	if parts[0] != "sea" || (parts[1] != "live" && parts[1] != "test") {
		return false // Invalid prefix
	}

	secret := parts[2]
	providedChecksum := parts[3]

	if len(providedChecksum) != apiKeyChecksumLength {
		return false
	}

	// Verify checksum
	expectedChecksum := s.generateChecksum(secret)
	return expectedChecksum == providedChecksum
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

// hashKey creates an HMAC-SHA256 hash of the API key.
func (s *APIKeyService) hashKey(key string) string {
	h := hmac.New(sha256.New, s.hmacSecret)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

// verifyKey verifies an API key against its stored hash.
func (s *APIKeyService) verifyKey(key, storedHash string) bool {
	computedHash := s.hashKey(key)
	return computedHash == storedHash
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
