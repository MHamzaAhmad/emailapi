package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// UsageCacheInterface defines the interface for usage tracking.
//
//go:generate mockgen -destination=mocks/mock_usage_cache.go -package=mocks github.com/emailapi/api/internal/repository/redis UsageCacheInterface
type UsageCacheInterface interface {
	// GetDailyUsage returns current day's email count for user.
	GetDailyUsage(ctx context.Context, userID string) (int64, error)

	// GetMonthlyUsage returns current month's email count for user.
	GetMonthlyUsage(ctx context.Context, userID string) (int64, error)

	// IncrementUsage atomically increments both daily and monthly counters.
	IncrementUsage(ctx context.Context, userID string, count int64) error

	// GetPlanState returns cached user plan state for fast-path checking.
	GetPlanState(ctx context.Context, userID string) (*PlanState, error)

	// SetPlanState caches user plan state.
	SetPlanState(ctx context.Context, userID string, state *PlanState) error

	// InvalidatePlanState removes cached state (on upgrade).
	InvalidatePlanState(ctx context.Context, userID string) error
}

// PlanState represents cached user plan/usage for fast checking.
type PlanState struct {
	Plan         string `json:"plan"`
	MonthlyLimit int64  `json:"monthly_limit"` // -1 = unlimited
	DailyLimit   int64  `json:"daily_limit"`   // -1 = no daily limit
	HardLimit    bool   `json:"hard_limit"`
	BillAllUsage bool   `json:"bill_all_usage"`
	MonthlyUsage int64  `json:"monthly_usage"`
	DailyUsage   int64  `json:"daily_usage"`
	CachedAt     int64  `json:"cached_at"` // Unix timestamp
}

// IsWithinDailyLimit checks if user can send more emails today.
func (s *PlanState) IsWithinDailyLimit(emailCount int64) bool {
	if s.DailyLimit < 0 {
		return true // No daily limit
	}
	return s.DailyUsage+emailCount <= s.DailyLimit
}

// IsWithinMonthlyLimit checks if user can send more emails this month.
func (s *PlanState) IsWithinMonthlyLimit(emailCount int64) bool {
	if s.MonthlyLimit < 0 {
		return true // Unlimited
	}
	return s.MonthlyUsage+emailCount <= s.MonthlyLimit
}

// RemainingDaily returns emails left today.
func (s *PlanState) RemainingDaily() int64 {
	if s.DailyLimit < 0 {
		return -1
	}
	remaining := s.DailyLimit - s.DailyUsage
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RemainingMonthly returns emails left this month.
func (s *PlanState) RemainingMonthly() int64 {
	if s.MonthlyLimit < 0 {
		return -1
	}
	remaining := s.MonthlyLimit - s.MonthlyUsage
	if remaining < 0 {
		return 0
	}
	return remaining
}

// UsageCache implements UsageCacheInterface.
type UsageCache struct {
	client *Client
}

// NewUsageCache creates a new UsageCache.
func NewUsageCache(client *Client) *UsageCache {
	return &UsageCache{client: client}
}

// dailyKey returns Redis key for daily counter: usage:daily:{userID}:{YYYYMMDD}
func (c *UsageCache) dailyKey(userID string) string {
	return fmt.Sprintf("usage:daily:%s:%s", userID, time.Now().UTC().Format("20060102"))
}

// monthlyKey returns Redis key for monthly counter: usage:monthly:{userID}:{YYYYMM}
func (c *UsageCache) monthlyKey(userID string) string {
	return fmt.Sprintf("usage:monthly:%s:%s", userID, time.Now().UTC().Format("200601"))
}

// stateKey returns Redis key for plan state: plan_state:{userID}
func (c *UsageCache) stateKey(userID string) string {
	return fmt.Sprintf("plan_state:%s", userID)
}

// GetDailyUsage returns current day's email count for user.
func (c *UsageCache) GetDailyUsage(ctx context.Context, userID string) (int64, error) {
	val, err := c.client.rdb.Get(ctx, c.dailyKey(userID)).Int64()
	if err == redis.Nil {
		return 0, nil // No usage today
	}
	return val, err
}

// GetMonthlyUsage returns current month's email count for user.
func (c *UsageCache) GetMonthlyUsage(ctx context.Context, userID string) (int64, error) {
	val, err := c.client.rdb.Get(ctx, c.monthlyKey(userID)).Int64()
	if err == redis.Nil {
		return 0, nil // No usage this month
	}
	return val, err
}

// IncrementUsage atomically increments both daily and monthly counters.
func (c *UsageCache) IncrementUsage(ctx context.Context, userID string, count int64) error {
	pipe := c.client.rdb.Pipeline()

	// Increment daily counter
	dailyKey := c.dailyKey(userID)
	pipe.IncrBy(ctx, dailyKey, count)
	pipe.Expire(ctx, dailyKey, 48*time.Hour) // TTL: 48 hours

	// Increment monthly counter
	monthlyKey := c.monthlyKey(userID)
	pipe.IncrBy(ctx, monthlyKey, count)
	pipe.Expire(ctx, monthlyKey, 45*24*time.Hour) // TTL: 45 days

	_, err := pipe.Exec(ctx)
	return err
}

// GetPlanState returns cached user plan state for fast-path checking.
func (c *UsageCache) GetPlanState(ctx context.Context, userID string) (*PlanState, error) {
	data, err := c.client.rdb.Get(ctx, c.stateKey(userID)).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, err
	}
	var state PlanState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SetPlanState caches user plan state.
func (c *UsageCache) SetPlanState(ctx context.Context, userID string, state *PlanState) error {
	state.CachedAt = time.Now().Unix()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return c.client.rdb.Set(ctx, c.stateKey(userID), data, 5*time.Minute).Err()
}

// InvalidatePlanState removes cached state (on upgrade).
func (c *UsageCache) InvalidatePlanState(ctx context.Context, userID string) error {
	return c.client.rdb.Del(ctx, c.stateKey(userID)).Err()
}

// Ensure concrete type implements interface
var _ UsageCacheInterface = (*UsageCache)(nil)
