package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CreditCacheInterface defines the interface for Polar credit caching.
//
//go:generate mockgen -destination=mocks/mock_credit_cache.go -package=mocks github.com/emailapi/api/internal/repository/redis CreditCacheInterface
type CreditCacheInterface interface {
	// GetState returns cached customer state (nil if cache miss).
	GetState(ctx context.Context, userID string) (*CachedCustomerState, error)

	// SetStateAndResetConsumed caches customer state and resets consumed counter.
	SetStateAndResetConsumed(ctx context.Context, userID string, state *CachedCustomerState) error

	// IncrConsumed increments the consumed counter for today.
	IncrConsumed(ctx context.Context, userID string, count int64) error

	// GetConsumed returns current consumed count for today.
	GetConsumed(ctx context.Context, userID string) (int64, error)

	// Invalidate removes all cached data for user.
	Invalidate(ctx context.Context, userID string) error
}

// CachedCustomerState represents cached Polar customer state.
type CachedCustomerState struct {
	IsPaid         bool   `json:"is_paid"`
	PlanType       string `json:"plan_type"`
	PolarBalance   int64  `json:"polar_balance"`
	PolarBalanceAt int64  `json:"polar_balance_at"` // Unix timestamp when balance was fetched
	ProductID      string `json:"product_id"`
}

// CreditCache implements CreditCacheInterface.
type CreditCache struct {
	client *Client
}

// NewCreditCache creates a new CreditCache.
func NewCreditCache(client *Client) *CreditCache {
	return &CreditCache{client: client}
}

// stateKey returns Redis key for customer state: polar:state:{userID}
func (c *CreditCache) stateKey(userID string) string {
	return fmt.Sprintf("polar:state:%s", userID)
}

// consumedKey returns Redis key for consumed counter: polar:consumed:{userID}:{YYYYMMDD}
func (c *CreditCache) consumedKey(userID string) string {
	return fmt.Sprintf("polar:consumed:%s:%s", userID, time.Now().UTC().Format("20060102"))
}

// GetState returns cached customer state (nil if cache miss).
func (c *CreditCache) GetState(ctx context.Context, userID string) (*CachedCustomerState, error) {
	data, err := c.client.rdb.Get(ctx, c.stateKey(userID)).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, err
	}
	var state CachedCustomerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SetStateAndResetConsumed caches customer state and resets consumed counter atomically.
func (c *CreditCache) SetStateAndResetConsumed(ctx context.Context, userID string, state *CachedCustomerState) error {
	state.PolarBalanceAt = time.Now().Unix()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	pipe := c.client.rdb.Pipeline()

	// Set state with 5 minute TTL
	pipe.Set(ctx, c.stateKey(userID), data, 5*time.Minute)

	// Reset consumed counter to 0
	consumedKey := c.consumedKey(userID)
	pipe.Set(ctx, consumedKey, 0, 48*time.Hour)

	_, err = pipe.Exec(ctx)
	return err
}

// IncrConsumed increments the consumed counter for today.
func (c *CreditCache) IncrConsumed(ctx context.Context, userID string, count int64) error {
	key := c.consumedKey(userID)
	pipe := c.client.rdb.Pipeline()
	pipe.IncrBy(ctx, key, count)
	pipe.Expire(ctx, key, 48*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// GetConsumed returns current consumed count for today.
func (c *CreditCache) GetConsumed(ctx context.Context, userID string) (int64, error) {
	val, err := c.client.rdb.Get(ctx, c.consumedKey(userID)).Int64()
	if err == redis.Nil {
		return 0, nil // No consumption today
	}
	return val, err
}

// Invalidate removes all cached data for user.
func (c *CreditCache) Invalidate(ctx context.Context, userID string) error {
	pipe := c.client.rdb.Pipeline()
	pipe.Del(ctx, c.stateKey(userID))
	pipe.Del(ctx, c.consumedKey(userID))
	_, err := pipe.Exec(ctx)
	return err
}

// Ensure concrete type implements interface
var _ CreditCacheInterface = (*CreditCache)(nil)
