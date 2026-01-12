package limit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing
type mockReputationCache struct {
	status *ReputationStatus
	err    error
}

func (m *mockReputationCache) Get(ctx context.Context, userID string) (*ReputationStatus, error) {
	return m.status, m.err
}

type mockCreditCache struct {
	state *CreditState
	err   error
}

func (m *mockCreditCache) GetState(ctx context.Context, userID string) (*CreditState, error) {
	return m.state, m.err
}

type mockUsageCache struct {
	dailyUsage int64
	err        error
}

func (m *mockUsageCache) GetDailyUsage(ctx context.Context, userID string) (int64, error) {
	return m.dailyUsage, m.err
}

func (m *mockUsageCache) IncrementDailyUsage(ctx context.Context, userID string, count int64) error {
	return nil
}

// Suspension Checker Tests

func TestSuspensionChecker_HardSuspended(t *testing.T) {
	checker := NewSuspensionChecker(&mockReputationCache{
		status: &ReputationStatus{IsHardSuspended: true},
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, ReasonHardSuspended, result.Reason)
}

func TestSuspensionChecker_SoftSuspended(t *testing.T) {
	checker := NewSuspensionChecker(&mockReputationCache{
		status: &ReputationStatus{IsSoftSuspended: true},
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed) // Soft suspended is allowed but with reduced limits
	assert.Equal(t, "soft", result.Meta["suspension_type"])
}

func TestSuspensionChecker_NotSuspended(t *testing.T) {
	checker := NewSuspensionChecker(&mockReputationCache{
		status: &ReputationStatus{},
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

// Daily Quota Checker Tests

func TestDailyQuotaChecker_FreeUserUnderLimit(t *testing.T) {
	checker := NewDailyQuotaChecker(DailyQuotaConfig{
		UsageCache:  &mockUsageCache{dailyUsage: 50},
		CreditCache: &mockCreditCache{state: &CreditState{IsPaid: false}},
		FreeLimit:   100,
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "50", result.Meta["daily_remaining"])
}

func TestDailyQuotaChecker_FreeUserAtLimit(t *testing.T) {
	checker := NewDailyQuotaChecker(DailyQuotaConfig{
		UsageCache:  &mockUsageCache{dailyUsage: 100},
		CreditCache: &mockCreditCache{state: &CreditState{IsPaid: false}},
		FreeLimit:   100,
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, ReasonDailyExceeded, result.Reason)
}

func TestDailyQuotaChecker_PaidUserNoLimit(t *testing.T) {
	checker := NewDailyQuotaChecker(DailyQuotaConfig{
		UsageCache:  &mockUsageCache{dailyUsage: 10000},
		CreditCache: &mockCreditCache{state: &CreditState{IsPaid: true}},
		FreeLimit:   100,
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "-1", result.Meta["daily_limit"])
}

func TestDailyQuotaChecker_SoftSuspendedReducedLimit(t *testing.T) {
	// Soft suspended gets 10% of limit (10 instead of 100)
	checker := NewDailyQuotaChecker(DailyQuotaConfig{
		UsageCache:  &mockUsageCache{dailyUsage: 10},
		CreditCache: &mockCreditCache{state: &CreditState{IsPaid: false}},
		RepCache:    &mockReputationCache{status: &ReputationStatus{IsSoftSuspended: true}},
		FreeLimit:   100,
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.False(t, result.Allowed) // At reduced limit (10)
	assert.Equal(t, ReasonDailyExceeded, result.Reason)
}

// Monthly Quota Checker Tests

func TestMonthlyQuotaChecker_FreeUserWithCredits(t *testing.T) {
	checker := NewMonthlyQuotaChecker(&mockCreditCache{
		state: &CreditState{IsPaid: false, PolarBalance: 1000},
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "1000", result.Meta["monthly_remaining"])
}

func TestMonthlyQuotaChecker_FreeUserNoCredits(t *testing.T) {
	checker := NewMonthlyQuotaChecker(&mockCreditCache{
		state: &CreditState{IsPaid: false, PolarBalance: 0},
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, ReasonCreditsExhausted, result.Reason)
}

func TestMonthlyQuotaChecker_PaidUserNoLimit(t *testing.T) {
	checker := NewMonthlyQuotaChecker(&mockCreditCache{
		state: &CreditState{IsPaid: true},
	})

	result, err := checker.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "-1", result.Meta["monthly_limit"])
}
