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

// ClientConfig holds configuration for the Redis client.
type ClientConfig struct {
	URL          string
	PoolSize     int
	MinIdleConns int
}

// NewClient creates a new Redis client with explicit pool configuration.
func NewClient(cfg ClientConfig) (*Client, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, err
	}

	opts.PoolSize = cfg.PoolSize
	opts.MinIdleConns = cfg.MinIdleConns

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

// =============================================================================
// Redis Consumer Groups (for at-least-once delivery)
// =============================================================================

// XGroupCreateMkStream creates a consumer group, creating the stream if needed.
// Uses MKSTREAM option to create stream if it doesn't exist.
// Returns nil if group already exists (ignores BUSYGROUP error).
func (c *Client) XGroupCreateMkStream(ctx context.Context, stream, group, start string) error {
	err := c.rdb.XGroupCreateMkStream(ctx, stream, group, start).Err()
	if err != nil && err.Error() == "BUSYGROUP Consumer Group name already exists" {
		return nil // Group already exists, that's fine
	}
	return err
}

// XReadGroup reads entries from a stream using consumer groups.
// Returns pending entries first (if id is "0"), then new entries (if id is ">").
// block=0 means no blocking; use a duration to block for new entries.
func (c *Client) XReadGroup(ctx context.Context, group, consumer, stream, id string, count int64, block time.Duration) ([]redis.XStream, error) {
	return c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, id},
		Count:    count,
		Block:    block,
	}).Result()
}

// XAck acknowledges messages as processed, removing them from the pending list.
func (c *Client) XAck(ctx context.Context, stream, group string, ids ...string) (int64, error) {
	return c.rdb.XAck(ctx, stream, group, ids...).Result()
}

// Underlying returns the raw redis client for advanced operations.
func (c *Client) Underlying() *redis.Client {
	return c.rdb
}

// =============================================================================
// Bloom Filter Operations (requires RedisBloom module)
// =============================================================================

// BFReserve creates a Bloom filter with specified error rate and capacity.
// This should be called once when initializing the filter.
func (c *Client) BFReserve(ctx context.Context, key string, errorRate float64, capacity int64) error {
	return c.rdb.Do(ctx, "BF.RESERVE", key, errorRate, capacity).Err()
}

// BFAdd adds an item to a Bloom filter.
// Creates the filter with default parameters if it doesn't exist.
func (c *Client) BFAdd(ctx context.Context, key string, item string) (bool, error) {
	result, err := c.rdb.Do(ctx, "BF.ADD", key, item).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// BFMAdd adds multiple items to a Bloom filter.
// Returns a slice of booleans indicating if each item was newly added.
func (c *Client) BFMAdd(ctx context.Context, key string, items ...string) ([]bool, error) {
	args := make([]interface{}, 0, len(items)+2)
	args = append(args, "BF.MADD", key)
	for _, item := range items {
		args = append(args, item)
	}

	result, err := c.rdb.Do(ctx, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	bools := make([]bool, len(result))
	for i, v := range result {
		bools[i] = v == 1
	}
	return bools, nil
}

// BFExists checks if an item exists in a Bloom filter.
// Returns true if the item might exist (possible false positive).
// Returns false if the item definitely does not exist.
func (c *Client) BFExists(ctx context.Context, key string, item string) (bool, error) {
	result, err := c.rdb.Do(ctx, "BF.EXISTS", key, item).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// BFMExists checks if multiple items exist in a Bloom filter.
// Returns a slice of booleans for each item.
func (c *Client) BFMExists(ctx context.Context, key string, items ...string) ([]bool, error) {
	args := make([]interface{}, 0, len(items)+2)
	args = append(args, "BF.MEXISTS", key)
	for _, item := range items {
		args = append(args, item)
	}

	result, err := c.rdb.Do(ctx, args...).Int64Slice()
	if err != nil {
		return nil, err
	}

	bools := make([]bool, len(result))
	for i, v := range result {
		bools[i] = v == 1
	}
	return bools, nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
