package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/emailapi/api/internal/domain"
)

// DomainCache handles caching for domain entities.
// All key management is centralized here.
type DomainCache struct {
	client *Client
	ttl    time.Duration
}

// NewDomainCache creates a new DomainCache.
func NewDomainCache(client *Client, ttl time.Duration) *DomainCache {
	return &DomainCache{
		client: client,
		ttl:    ttl,
	}
}

// Key patterns - centralized in one place
func keyDomain(id string) string            { return "domain:" + id }
func keyDomainsByUser(userID string) string { return "domains:user:" + userID }

// GetByID retrieves a cached domain by ID.
// Returns nil, nil if not found in cache.
func (c *DomainCache) GetByID(ctx context.Context, id string) (*domain.SendingDomain, error) {
	data, err := c.client.Get(ctx, keyDomain(id))
	if err != nil {
		return nil, nil // Cache miss
	}

	var d domain.SendingDomain
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &d, nil
}

// SetByID caches a domain by ID.
func (c *DomainCache) SetByID(ctx context.Context, d *domain.SendingDomain) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyDomain(d.ID), string(data), c.ttl)
}

// GetByUserID retrieves cached domains list for a user.
// Returns nil, nil if not found in cache.
func (c *DomainCache) GetByUserID(ctx context.Context, userID string) ([]*domain.SendingDomain, error) {
	data, err := c.client.Get(ctx, keyDomainsByUser(userID))
	if err != nil {
		return nil, nil // Cache miss
	}

	var domains []*domain.SendingDomain
	if err := json.Unmarshal([]byte(data), &domains); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return domains, nil
}

// SetByUserID caches domains list for a user.
func (c *DomainCache) SetByUserID(ctx context.Context, userID string, domains []*domain.SendingDomain) error {
	data, err := json.Marshal(domains)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyDomainsByUser(userID), string(data), c.ttl)
}

// InvalidateByID removes a domain from cache by ID.
func (c *DomainCache) InvalidateByID(ctx context.Context, id string) error {
	return c.client.Del(ctx, keyDomain(id))
}

// InvalidateByUserID removes the domains list cache for a user.
func (c *DomainCache) InvalidateByUserID(ctx context.Context, userID string) error {
	return c.client.Del(ctx, keyDomainsByUser(userID))
}

// InvalidateAll removes both domain and user list cache.
// Use on Create/Update/Delete operations.
func (c *DomainCache) InvalidateAll(ctx context.Context, id, userID string) error {
	return c.client.Del(ctx, keyDomain(id), keyDomainsByUser(userID))
}
