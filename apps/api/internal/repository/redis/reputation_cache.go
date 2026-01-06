package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// UserReputationStatus represents cached reputation status for fast lookups.
type UserReputationStatus struct {
	IsFlagged   bool    `json:"is_flagged"`
	IsSuspended bool    `json:"is_suspended"`
	Score       float64 `json:"score"`
	UpdatedAt   int64   `json:"updated_at"` // Unix timestamp
}

// ReputationCache provides high-performance caching for user reputation status.
// Uses a short TTL (30s) to ensure changes propagate quickly while still
// providing significant performance benefits for the email sending hot path.
type ReputationCache struct {
	client *Client
	ttl    time.Duration
}

// NewReputationCache creates a new ReputationCache.
func NewReputationCache(client *Client) *ReputationCache {
	return &ReputationCache{
		client: client,
		ttl:    30 * time.Second, // Short TTL for quick propagation
	}
}

// reputationKey returns the Redis key for user reputation status.
func reputationKey(userID string) string {
	return fmt.Sprintf("reputation:status:%s", userID)
}

// Get retrieves cached reputation status for a user.
// Returns nil if not cached (cache miss).
func (c *ReputationCache) Get(ctx context.Context, userID string) (*UserReputationStatus, error) {
	key := reputationKey(userID)

	data, err := c.client.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get reputation cache: %w", err)
	}

	var status UserReputationStatus
	if err := json.Unmarshal(data, &status); err != nil {
		// Invalid cache entry, treat as miss
		return nil, nil
	}

	return &status, nil
}

// Set caches reputation status for a user.
func (c *ReputationCache) Set(ctx context.Context, userID string, status *UserReputationStatus) error {
	key := reputationKey(userID)

	status.UpdatedAt = time.Now().Unix()

	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal reputation status: %w", err)
	}

	if err := c.client.rdb.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set reputation cache: %w", err)
	}

	return nil
}

// Delete invalidates the cached reputation status for a user.
// Should be called when reputation is updated (suspend/unsuspend/flag changes).
func (c *ReputationCache) Delete(ctx context.Context, userID string) error {
	key := reputationKey(userID)

	if err := c.client.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete reputation cache: %w", err)
	}

	return nil
}

// SetSuspended is a convenience method to quickly mark a user as suspended in cache.
// This is useful for immediate blocking after auto-suspension.
func (c *ReputationCache) SetSuspended(ctx context.Context, userID string) error {
	return c.Set(ctx, userID, &UserReputationStatus{
		IsSuspended: true,
		IsFlagged:   true,
	})
}
