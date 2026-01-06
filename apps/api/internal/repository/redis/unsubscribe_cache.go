package redis

import (
	"context"
	"fmt"
	"time"
)

const (
	// Key format: unsubscribe:{user_id}:{email_hash}
	unsubscribeKeyPrefix = "unsubscribe:"
	// Default TTL for unsubscribe cache entries (24 hours)
	unsubscribeCacheTTL = 24 * time.Hour
)

// UnsubscribeCache provides fast lookups for unsubscribed emails using Redis.
// Uses a 24-hour TTL with passive refresh on reads.
type UnsubscribeCache struct {
	client *Client
	ttl    time.Duration
}

// NewUnsubscribeCache creates a new UnsubscribeCache.
func NewUnsubscribeCache(client *Client) *UnsubscribeCache {
	return &UnsubscribeCache{
		client: client,
		ttl:    unsubscribeCacheTTL,
	}
}

// unsubscribeKey returns the Redis key for a user+email unsubscribe entry.
func unsubscribeKey(userID, emailHash string) string {
	return fmt.Sprintf("%s%s:%s", unsubscribeKeyPrefix, userID, emailHash)
}

// Set marks an email as unsubscribed in cache.
func (c *UnsubscribeCache) Set(ctx context.Context, userID, emailHash string) error {
	key := unsubscribeKey(userID, emailHash)
	if err := c.client.Set(ctx, key, "1", c.ttl); err != nil {
		return fmt.Errorf("failed to set unsubscribe cache: %w", err)
	}
	return nil
}

// Check returns true if the email is unsubscribed (exists in cache).
// Returns (found, error) - found=true means unsubscribed, found=false means not in cache.
func (c *UnsubscribeCache) Check(ctx context.Context, userID, emailHash string) (bool, error) {
	key := unsubscribeKey(userID, emailHash)
	val, err := c.client.rdb.Get(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil // Not in cache (cache miss)
		}
		return false, fmt.Errorf("failed to check unsubscribe cache: %w", err)
	}
	return val == "1", nil
}

// CheckBatch checks multiple email hashes for a user using MGET for O(1) batch lookups.
// Returns the list of email hashes that are unsubscribed (found in cache).
func (c *UnsubscribeCache) CheckBatch(ctx context.Context, userID string, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	// Build Redis keys
	keys := make([]string, len(hashes))
	for i, hash := range hashes {
		keys[i] = unsubscribeKey(userID, hash)
	}

	// Single MGET call for all hashes
	values, err := c.client.MGet(ctx, keys...)
	if err != nil {
		return nil, fmt.Errorf("redis mget failed: %w", err)
	}

	// Collect unsubscribed hashes
	var unsubscribed []string
	for i, val := range values {
		if val != nil {
			unsubscribed = append(unsubscribed, hashes[i])
		}
	}

	return unsubscribed, nil
}

// SetBatch marks multiple emails as unsubscribed in cache (pipeline for efficiency).
func (c *UnsubscribeCache) SetBatch(ctx context.Context, userID string, hashes []string) error {
	if len(hashes) == 0 {
		return nil
	}

	pipe := c.client.rdb.Pipeline()
	for _, hash := range hashes {
		key := unsubscribeKey(userID, hash)
		pipe.Set(ctx, key, "1", c.ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set batch unsubscribe cache: %w", err)
	}

	return nil
}

// Delete removes an unsubscribe entry from cache (for resubscribe).
func (c *UnsubscribeCache) Delete(ctx context.Context, userID, emailHash string) error {
	key := unsubscribeKey(userID, emailHash)
	if err := c.client.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete unsubscribe cache: %w", err)
	}
	return nil
}
