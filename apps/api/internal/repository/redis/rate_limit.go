package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter provides high-performance rate limiting using Redis.
// Uses sliding window counter algorithm with atomic Lua scripts for O(1) operations.
type RateLimiter struct {
	client *Client
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(client *Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// RateLimitResult contains the result of a rate limit check.
type RateLimitResult struct {
	Allowed   bool      // Whether the request is allowed
	Remaining int       // Remaining requests in the window
	ResetAt   time.Time // When the window resets
}

// slidingWindowScript is a Lua script for atomic sliding window rate limiting.
// KEYS[1] = rate limit key
// ARGV[1] = limit (max requests)
// ARGV[2] = window in milliseconds
// ARGV[3] = current timestamp in milliseconds
// Returns: [current_count, ttl_ms]
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])

local current = redis.call('INCR', key)
if current == 1 then
    redis.call('PEXPIRE', key, window_ms)
end

local ttl = redis.call('PTTL', key)
if ttl < 0 then
    ttl = window_ms
end

return {current, ttl}
`)

// Check performs a sliding window rate limit check.
// Returns RateLimitResult with whether the request is allowed, remaining quota, and reset time.
func (r *RateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	windowMs := window.Milliseconds()
	nowMs := time.Now().UnixMilli()

	result, err := slidingWindowScript.Run(ctx, r.client.rdb, []string{key}, limit, windowMs, nowMs).Slice()
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	if len(result) != 2 {
		return nil, fmt.Errorf("unexpected script result length: %d", len(result))
	}

	current, err := toInt64(result[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse current count: %w", err)
	}

	ttlMs, err := toInt64(result[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse TTL: %w", err)
	}

	allowed := current <= int64(limit)
	remaining := limit - int(current)
	if remaining < 0 {
		remaining = 0
	}

	resetAt := time.Now().Add(time.Duration(ttlMs) * time.Millisecond)

	return &RateLimitResult{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}

// streamCounterScript atomically increments stream count and checks limit.
// KEYS[1] = stream counter key
// ARGV[1] = max streams allowed
// ARGV[2] = TTL in seconds (safety net for cleanup)
// Returns: [new_count, allowed (1 or 0)]
var streamCounterScript = redis.NewScript(`
local key = KEYS[1]
local max_streams = tonumber(ARGV[1])
local ttl_seconds = tonumber(ARGV[2])

local current = redis.call('GET', key)
if current == false then
    current = 0
else
    current = tonumber(current)
end

if current >= max_streams then
    return {current, 0}
end

local new_count = redis.call('INCR', key)
redis.call('EXPIRE', key, ttl_seconds)
return {new_count, 1}
`)

// IncrementStreams atomically increments the concurrent stream count for a user.
// Returns whether the new stream is allowed (under the limit).
// The counter has a TTL as a safety net in case streams aren't properly decremented.
func (r *RateLimiter) IncrementStreams(ctx context.Context, userID string, maxStreams int) (bool, error) {
	key := streamKey(userID)
	// 1 hour TTL as safety net - streams should be decremented on disconnect
	ttlSeconds := 3600

	result, err := streamCounterScript.Run(ctx, r.client.rdb, []string{key}, maxStreams, ttlSeconds).Slice()
	if err != nil {
		return false, fmt.Errorf("increment streams failed: %w", err)
	}

	if len(result) != 2 {
		return false, fmt.Errorf("unexpected script result length: %d", len(result))
	}

	allowed, err := toInt64(result[1])
	if err != nil {
		return false, fmt.Errorf("failed to parse allowed: %w", err)
	}

	return allowed == 1, nil
}

// DecrementStreams atomically decrements the concurrent stream count for a user.
// Ensures the count doesn't go below 0.
func (r *RateLimiter) DecrementStreams(ctx context.Context, userID string) error {
	key := streamKey(userID)

	// Use Lua script to ensure we don't go below 0
	script := redis.NewScript(`
		local key = KEYS[1]
		local current = redis.call('GET', key)
		if current == false or tonumber(current) <= 0 then
			return 0
		end
		return redis.call('DECR', key)
	`)

	_, err := script.Run(ctx, r.client.rdb, []string{key}).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("decrement streams failed: %w", err)
	}

	return nil
}

// GetStreamCount returns the current concurrent stream count for a user.
func (r *RateLimiter) GetStreamCount(ctx context.Context, userID string) (int, error) {
	key := streamKey(userID)
	result, err := r.client.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get stream count failed: %w", err)
	}

	count, err := strconv.Atoi(result)
	if err != nil {
		return 0, fmt.Errorf("parse stream count failed: %w", err)
	}

	return count, nil
}

// streamKey returns the Redis key for concurrent stream tracking.
func streamKey(userID string) string {
	return fmt.Sprintf("ratelimit:streams:%s", userID)
}

// RateLimitKey returns the Redis key for rate limiting a user.
func RateLimitKey(userID string) string {
	return fmt.Sprintf("ratelimit:req:%s", userID)
}

// toInt64 converts an interface{} to int64, handling both int64 and string types.
func toInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		return 0, fmt.Errorf("unexpected type %T", v)
	}
}
