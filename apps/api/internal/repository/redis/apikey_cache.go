package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/emailapi/api/internal/domain"
)

// APIKeyCache handles caching for API key entities.
// All key management is centralized here.
type APIKeyCache struct {
	client *Client
	ttl    time.Duration
}

// NewAPIKeyCache creates a new APIKeyCache.
func NewAPIKeyCache(client *Client, ttl time.Duration) *APIKeyCache {
	return &APIKeyCache{
		client: client,
		ttl:    ttl,
	}
}

// Key patterns - centralized in one place
func keyAPIKey(id string) string            { return "apikey:" + id }
func keyAPIKeyPrefix(prefix string) string  { return "apikey:prefix:" + prefix }
func keyAPIKeyHash(keyHash string) string   { return "apikey:hash:" + keyHash }
func keyAPIKeysByUser(userID string) string { return "apikeys:user:" + userID }

// GetByID retrieves a cached API key by ID.
// Returns nil, nil if not found in cache.
func (c *APIKeyCache) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	data, err := c.client.Get(ctx, keyAPIKey(id))
	if err != nil {
		return nil, nil // Cache miss
	}

	var k domain.APIKey
	if err := json.Unmarshal([]byte(data), &k); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &k, nil
}

// SetByID caches an API key by ID.
func (c *APIKeyCache) SetByID(ctx context.Context, k *domain.APIKey) error {
	data, err := json.Marshal(k)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyAPIKey(k.ID), string(data), c.ttl)
}

// GetByPrefix retrieves a cached API key by prefix (for auth lookup).
// Returns nil, nil if not found in cache.
func (c *APIKeyCache) GetByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error) {
	data, err := c.client.Get(ctx, keyAPIKeyPrefix(prefix))
	if err != nil {
		return nil, nil // Cache miss
	}

	var k domain.APIKey
	if err := json.Unmarshal([]byte(data), &k); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &k, nil
}

// SetByPrefix caches an API key by prefix (for auth lookup).
func (c *APIKeyCache) SetByPrefix(ctx context.Context, k *domain.APIKey) error {
	data, err := json.Marshal(k)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyAPIKeyPrefix(k.KeyPrefix), string(data), c.ttl)
}

// GetByUserID retrieves cached API keys list for a user.
// Returns nil, nil if not found in cache.
func (c *APIKeyCache) GetByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	data, err := c.client.Get(ctx, keyAPIKeysByUser(userID))
	if err != nil {
		return nil, nil // Cache miss
	}

	var keys []*domain.APIKey
	if err := json.Unmarshal([]byte(data), &keys); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return keys, nil
}

// SetByUserID caches API keys list for a user.
func (c *APIKeyCache) SetByUserID(ctx context.Context, userID string, keys []*domain.APIKey) error {
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyAPIKeysByUser(userID), string(data), c.ttl)
}

// InvalidateByID removes an API key from cache by ID.
func (c *APIKeyCache) InvalidateByID(ctx context.Context, id string) error {
	return c.client.Del(ctx, keyAPIKey(id))
}

// InvalidateByPrefix removes an API key from cache by prefix.
func (c *APIKeyCache) InvalidateByPrefix(ctx context.Context, prefix string) error {
	return c.client.Del(ctx, keyAPIKeyPrefix(prefix))
}

// InvalidateByUserID removes the API keys list cache for a user.
func (c *APIKeyCache) InvalidateByUserID(ctx context.Context, userID string) error {
	return c.client.Del(ctx, keyAPIKeysByUser(userID))
}

// InvalidateAll removes all cache entries for an API key.
// Use on Update/Delete/Revoke operations.
func (c *APIKeyCache) InvalidateAll(ctx context.Context, id, prefix, userID string) error {
	return c.client.Del(ctx, keyAPIKey(id), keyAPIKeyPrefix(prefix), keyAPIKeysByUser(userID))
}

// GetByKeyHash retrieves a cached API key by its hash (for fast-path auth).
// Returns nil, nil if not found in cache.
func (c *APIKeyCache) GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	data, err := c.client.Get(ctx, keyAPIKeyHash(keyHash))
	if err != nil {
		return nil, nil // Cache miss
	}

	var k domain.APIKey
	if err := json.Unmarshal([]byte(data), &k); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &k, nil
}

// SetByKeyHash caches an API key by its hash (for fast-path auth).
func (c *APIKeyCache) SetByKeyHash(ctx context.Context, keyHash string, k *domain.APIKey) error {
	data, err := json.Marshal(k)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyAPIKeyHash(keyHash), string(data), c.ttl)
}

// InvalidateByKeyHash removes an API key from cache by its hash.
func (c *APIKeyCache) InvalidateByKeyHash(ctx context.Context, keyHash string) error {
	return c.client.Del(ctx, keyAPIKeyHash(keyHash))
}
