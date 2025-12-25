package redis

import (
	"context"
	"time"
)

const (
	// mxCachePrefix is the key prefix for MX record cache entries.
	mxCachePrefix = "mx:"

	// mxCacheTTL is the TTL for MX record cache entries.
	// 1 hour is reasonable as MX records rarely change.
	mxCacheTTL = 1 * time.Hour
)

// MXCache caches MX record lookup results to avoid repeated DNS lookups.
// This significantly speeds up recipient validation for repeated sends.
type MXCache struct {
	client *Client
}

// NewMXCache creates a new MXCache.
func NewMXCache(client *Client) *MXCache {
	return &MXCache{client: client}
}

// keyMX generates the cache key for a domain.
func keyMX(domain string) string {
	return mxCachePrefix + domain
}

// HasMX checks if a domain has MX records (cached).
// Returns:
//   - (*true, nil) if domain has MX records
//   - (*false, nil) if domain does NOT have MX records
//   - (nil, nil) on cache miss
//   - (nil, error) on Redis error
func (c *MXCache) HasMX(ctx context.Context, domain string) (*bool, error) {
	val, err := c.client.Get(ctx, keyMX(domain))
	if err != nil {
		// Cache miss - key doesn't exist
		return nil, nil
	}

	hasMX := val == "1"
	return &hasMX, nil
}

// SetMX caches whether a domain has MX records.
func (c *MXCache) SetMX(ctx context.Context, domain string, hasMX bool) error {
	val := "0"
	if hasMX {
		val = "1"
	}
	return c.client.Set(ctx, keyMX(domain), val, mxCacheTTL)
}

// MXCacheInterface defines the interface for MX record caching.
type MXCacheInterface interface {
	HasMX(ctx context.Context, domain string) (*bool, error)
	SetMX(ctx context.Context, domain string, hasMX bool) error
}

// Ensure MXCache implements MXCacheInterface
var _ MXCacheInterface = (*MXCache)(nil)
