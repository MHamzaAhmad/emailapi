package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/emailapi/api/internal/domain"
)

// UserCache handles caching for user entities.
// All key management is centralized here.
type UserCache struct {
	client *Client
	ttl    time.Duration
}

// NewUserCache creates a new UserCache.
func NewUserCache(client *Client, ttl time.Duration) *UserCache {
	return &UserCache{
		client: client,
		ttl:    ttl,
	}
}

// Key patterns - centralized in one place
func keyUser(id string) string                   { return "user:" + id }
func keyUserEmail(email string) string           { return "user:email:" + email }
func keyUserExternalID(externalID string) string { return "user:external_id:" + externalID }

// GetByID retrieves a cached user by ID.
// Returns nil, nil if not found in cache.
func (c *UserCache) GetByID(ctx context.Context, id string) (*domain.User, error) {
	data, err := c.client.Get(ctx, keyUser(id))
	if err != nil {
		return nil, nil // Cache miss
	}

	var u domain.User
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &u, nil
}

// SetByID caches a user by ID.
func (c *UserCache) SetByID(ctx context.Context, u *domain.User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyUser(u.ID), string(data), c.ttl)
}

// GetByEmail retrieves a cached user by email.
// Returns nil, nil if not found in cache.
func (c *UserCache) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	data, err := c.client.Get(ctx, keyUserEmail(email))
	if err != nil {
		return nil, nil // Cache miss
	}

	var u domain.User
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &u, nil
}

// SetByEmail caches a user by email.
func (c *UserCache) SetByEmail(ctx context.Context, u *domain.User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyUserEmail(u.Email), string(data), c.ttl)
}

// InvalidateByID removes a user from cache by ID.
func (c *UserCache) InvalidateByID(ctx context.Context, id string) error {
	return c.client.Del(ctx, keyUser(id))
}

// InvalidateByEmail removes a user from cache by email.
func (c *UserCache) InvalidateByEmail(ctx context.Context, email string) error {
	return c.client.Del(ctx, keyUserEmail(email))
}

// InvalidateAll removes all cache entries for a user.
// Use on Update/Delete operations.
func (c *UserCache) InvalidateAll(ctx context.Context, id, email string) error {
	return c.client.Del(ctx, keyUser(id), keyUserEmail(email))
}

// GetByExternalID retrieves a cached user by external ID (Clerk ID).
// Returns nil, nil if not found in cache.
func (c *UserCache) GetByExternalID(ctx context.Context, externalID string) (*domain.User, error) {
	data, err := c.client.Get(ctx, keyUserExternalID(externalID))
	if err != nil {
		return nil, nil // Cache miss
	}

	var u domain.User
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &u, nil
}

// SetByExternalID caches a user by external ID (Clerk ID).
func (c *UserCache) SetByExternalID(ctx context.Context, u *domain.User) error {
	if u.ExternalID == nil || *u.ExternalID == "" {
		return nil // Skip caching if no external ID
	}
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyUserExternalID(*u.ExternalID), string(data), c.ttl)
}

// InvalidateByExternalID removes a user from cache by external ID.
func (c *UserCache) InvalidateByExternalID(ctx context.Context, externalID string) error {
	return c.client.Del(ctx, keyUserExternalID(externalID))
}
