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
func keyDomainDetails(id string) string     { return "domain:details:" + id }
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

// GetDetailsByID retrieves cached domain with full details (including DNS record statuses).
// This is set after Verify() to preserve validated record statuses for subsequent Get()/List() calls.
// Returns nil, nil if not found in cache.
func (c *DomainCache) GetDetailsByID(ctx context.Context, id string) (*domain.DomainWithDetails, error) {
	data, err := c.client.Get(ctx, keyDomainDetails(id))
	if err != nil {
		return nil, nil // Cache miss
	}

	var d domain.DomainWithDetails
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &d, nil
}

// SetDetailsByID caches domain with full details (including DNS record statuses).
// Call this after Verify() to preserve validated record statuses for subsequent Get()/List() calls.
func (c *DomainCache) SetDetailsByID(ctx context.Context, d *domain.DomainWithDetails) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keyDomainDetails(d.ID), string(data), c.ttl)
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
	return c.client.Del(ctx, keyDomain(id), keyDomainDetails(id))
}

// InvalidateByUserID removes the domains list cache for a user.
func (c *DomainCache) InvalidateByUserID(ctx context.Context, userID string) error {
	return c.client.Del(ctx, keyDomainsByUser(userID))
}

// InvalidateAll removes domain, domain details, and user list cache.
// Use on Create/Update/Delete operations.
func (c *DomainCache) InvalidateAll(ctx context.Context, id, userID string) error {
	return c.client.Del(ctx, keyDomain(id), keyDomainDetails(id), keyDomainsByUser(userID))
}

// Fast-path key for sending validation lookups
func keySendingStatus(userID, domainName string) string {
	return "domain:sending:" + userID + ":" + domainName
}

// sendingStatusTTL is shorter than the main cache TTL for responsiveness to DNS changes.
const sendingStatusTTL = 5 * time.Minute

// GetSendingStatus retrieves a cached domain for sending validation.
// This is a high-frequency, low-latency path optimized for email sending.
// Returns nil, nil if not found in cache.
func (c *DomainCache) GetSendingStatus(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error) {
	data, err := c.client.Get(ctx, keySendingStatus(userID, domainName))
	if err != nil {
		return nil, nil // Cache miss
	}

	var d domain.SendingDomain
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return nil, nil // Invalid cache data, treat as miss
	}

	return &d, nil
}

// SetSendingStatus caches a domain for sending validation.
func (c *DomainCache) SetSendingStatus(ctx context.Context, d *domain.SendingDomain) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, keySendingStatus(d.UserID, d.Domain), string(data), sendingStatusTTL)
}

// InvalidateSendingStatus removes the sending status cache for a domain.
func (c *DomainCache) InvalidateSendingStatus(ctx context.Context, userID, domainName string) error {
	return c.client.Del(ctx, keySendingStatus(userID, domainName))
}
