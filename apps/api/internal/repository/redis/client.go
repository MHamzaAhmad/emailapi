package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client for cache operations.
type Client struct {
	rdb *redis.Client
}

// NewClient creates a new Redis client from the connection URL.
func NewClient(redisURL string) (*Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opts)

	return &Client{rdb: rdb}, nil
}

// Ping checks the Redis connection.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Set stores a key-value pair with optional TTL.
// If ttl is 0, the key will not expire.
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// MGet retrieves multiple values by keys in a single round-trip.
// Returns a slice where each element is the value or nil if not found.
func (c *Client) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return c.rdb.MGet(ctx, keys...).Result()
}

// Del deletes one or more keys.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// SetNX sets a key only if it doesn't exist (for deduplication).
func (c *Client) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, value, ttl).Result()
}

// =============================================================================
// Redis Streams Operations
// =============================================================================

// XAdd adds an entry to a stream and returns the generated ID.
func (c *Client) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	return c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
}

// XRead reads entries from streams with blocking support.
// Pass 0 for block to not block, or a duration to block for new entries.
func (c *Client) XRead(ctx context.Context, streams []string, count int64, block time.Duration) ([]redis.XStream, error) {
	return c.rdb.XRead(ctx, &redis.XReadArgs{
		Streams: streams,
		Count:   count,
		Block:   block,
	}).Result()
}

// XTrimMaxLenApprox trims a stream to approximately maxLen entries.
// Uses approximate trimming for efficiency.
func (c *Client) XTrimMaxLenApprox(ctx context.Context, stream string, maxLen int64) error {
	return c.rdb.XTrimMaxLenApprox(ctx, stream, maxLen, 0).Err()
}

// Underlying returns the raw redis client for advanced operations.
func (c *Client) Underlying() *redis.Client {
	return c.rdb
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
