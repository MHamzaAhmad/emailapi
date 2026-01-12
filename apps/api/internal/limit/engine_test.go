package limit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockChecker for testing
type mockChecker struct {
	name   string
	result *CheckResult
	err    error
	delay  time.Duration
}

func (m *mockChecker) Check(ctx context.Context, userID string) (*CheckResult, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return m.result, m.err
}

func (m *mockChecker) Name() string { return m.name }

func TestEngine_AllAllowed(t *testing.T) {
	engine := NewEngine(
		&mockChecker{name: "check1", result: Allowed()},
		&mockChecker{name: "check2", result: Allowed()},
		&mockChecker{name: "check3", result: Allowed()},
	)

	result, err := engine.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Empty(t, result.Reason)
}

func TestEngine_OneBlocked(t *testing.T) {
	engine := NewEngine(
		&mockChecker{name: "check1", result: Allowed()},
		&mockChecker{name: "blocked", result: Blocked(ReasonHardSuspended, PrioritySuspension)},
		&mockChecker{name: "check3", result: Allowed()},
	)

	result, err := engine.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, ReasonHardSuspended, result.Reason)
}

func TestEngine_PriorityOrder(t *testing.T) {
	// Multiple failures - highest priority (lowest number) wins
	engine := NewEngine(
		&mockChecker{name: "rate", result: Blocked(ReasonRateLimited, PriorityRateLimit)},
		&mockChecker{name: "suspend", result: Blocked(ReasonHardSuspended, PrioritySuspension)},
		&mockChecker{name: "daily", result: Blocked(ReasonDailyExceeded, PriorityDailyQuota)},
	)

	result, err := engine.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, ReasonHardSuspended, result.Reason) // Priority 1 wins
}

func TestEngine_ErrorFailsOpen(t *testing.T) {
	// Errors should fail open (allow request)
	engine := NewEngine(
		&mockChecker{name: "error", result: nil, err: errors.New("redis error")},
		&mockChecker{name: "ok", result: Allowed()},
	)

	result, err := engine.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestEngine_MetadataMerged(t *testing.T) {
	engine := NewEngine(
		&mockChecker{name: "check1", result: Allowed().WithMeta("key1", "val1")},
		&mockChecker{name: "check2", result: Allowed().WithMeta("key2", "val2")},
	)

	result, err := engine.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.Equal(t, "val1", result.Meta["key1"])
	assert.Equal(t, "val2", result.Meta["key2"])
}

func TestEngine_EmptyCheckers(t *testing.T) {
	engine := NewEngine()

	result, err := engine.Check(context.Background(), "user_1")

	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestEngine_ParallelExecution(t *testing.T) {
	// All checkers should run in parallel
	// Total time should be ~50ms, not 150ms if sequential
	engine := NewEngine(
		&mockChecker{name: "slow1", result: Allowed(), delay: 50 * time.Millisecond},
		&mockChecker{name: "slow2", result: Allowed(), delay: 50 * time.Millisecond},
		&mockChecker{name: "slow3", result: Allowed(), delay: 50 * time.Millisecond},
	)

	start := time.Now()
	result, err := engine.Check(context.Background(), "user_1")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	// Should complete in ~50ms (parallel), not ~150ms (sequential)
	assert.Less(t, elapsed, 100*time.Millisecond)
}
