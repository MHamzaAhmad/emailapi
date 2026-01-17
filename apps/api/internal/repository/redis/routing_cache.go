package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	routingKeyPrefix = "routing:"
	routingTTL       = 7 * 24 * time.Hour // 7 days
)

// RoutingEntry represents a cached routing entry for reply threading.
type RoutingEntry struct {
	MessageID string `json:"message_id"`
	UserID    string `json:"user_id"`
}

// RoutingCacheInterface defines the interface for email routing caching.
type RoutingCacheInterface interface {
	// Set stores a routing entry (email_id → message_id + user_id).
	Set(ctx context.Context, emailID, messageID, userID string) error
	// Get retrieves routing by email_id. Returns nil if not found.
	Get(ctx context.Context, emailID string) (*RoutingEntry, error)
}

// RoutingCache implements RoutingCacheInterface using Redis.
type RoutingCache struct {
	client redis.UniversalClient
}

// NewRoutingCache creates a new RoutingCache.
func NewRoutingCache(client redis.UniversalClient) *RoutingCache {
	return &RoutingCache{client: client}
}

// Set stores a routing entry.
func (c *RoutingCache) Set(ctx context.Context, emailID, messageID, userID string) error {
	entry := RoutingEntry{
		MessageID: messageID,
		UserID:    userID,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal routing entry: %w", err)
	}
	key := routingKeyPrefix + emailID
	return c.client.Set(ctx, key, data, routingTTL).Err()
}

// Get retrieves a routing entry by email_id.
func (c *RoutingCache) Get(ctx context.Context, emailID string) (*RoutingEntry, error) {
	key := routingKeyPrefix + emailID
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get routing entry: %w", err)
	}
	var entry RoutingEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal routing entry: %w", err)
	}
	return &entry, nil
}

// Ensure concrete type implements interface.
var _ RoutingCacheInterface = (*RoutingCache)(nil)
