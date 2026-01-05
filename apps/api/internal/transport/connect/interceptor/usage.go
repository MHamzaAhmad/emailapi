package interceptor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	"github.com/emailapi/api/internal/external/polar"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// UsageConfig holds configuration for the usage limit interceptor.
type UsageConfig struct {
	CreditCache redisrepo.CreditCacheInterface
	PolarClient polar.Client
	Enabled     bool
}

// usageProcedures defines which procedures count against usage limits.
var usageProcedures = map[string]bool{
	"/v1.EmailService/SendEmail": true,
}

type usageInterceptor struct {
	cfg UsageConfig
	// Debounce refresh requests per user
	refreshMu      sync.Mutex
	lastRefreshReq map[string]time.Time
}

// NewUsageInterceptor creates a new usage limit interceptor.
// Must be placed AFTER auth interceptor (needs user ID from context).
// Must be placed BEFORE rate limit interceptor (usage = billing, rate = abuse prevention).
func NewUsageInterceptor(cfg UsageConfig) connect.Interceptor {
	return &usageInterceptor{
		cfg:            cfg,
		lastRefreshReq: make(map[string]time.Time),
	}
}

// WrapUnary implements connect.Interceptor for unary calls.
func (u *usageInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if !u.cfg.Enabled {
			return next(ctx, req)
		}

		procedure := req.Spec().Procedure
		if !usageProcedures[procedure] {
			return next(ctx, req)
		}

		userID := GetUserID(ctx)
		if userID == "" {
			return next(ctx, req)
		}

		// Get customer state from cache (or fetch from Polar on miss)
		state, err := u.getOrFetchState(ctx, userID)
		if err != nil {
			// Fail open on error - allow request but log
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to get credit state, allowing request")
			return next(ctx, req)
		}

		// Paid users: allow immediately, ingest event async
		if state.IsPaid {
			resp, respErr := next(ctx, req)
			if respErr == nil && resp != nil {
				go u.postSendPaid(context.Background(), userID, 1)
			}
			return resp, respErr
		}

		// Free users: check credit balance before allowing
		consumed, err := u.cfg.CreditCache.GetConsumed(ctx, userID)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to get consumed count, allowing request")
			consumed = 0
		}

		effectiveBalance := state.PolarBalance - consumed
		if effectiveBalance <= 0 {
			return nil, creditExhaustedError(state.PolarBalance, consumed)
		}

		// Allow the email send
		resp, respErr := next(ctx, req)

		// Post-send: increment consumed and ingest event (async)
		if respErr == nil && resp != nil {
			go u.postSendFree(context.Background(), userID, 1)
		}

		return resp, respErr
	}
}

// getOrFetchState gets cached state or fetches from Polar on cache miss.
func (u *usageInterceptor) getOrFetchState(ctx context.Context, userID string) (*redisrepo.CachedCustomerState, error) {
	state, err := u.cfg.CreditCache.GetState(ctx, userID)
	if err != nil {
		return nil, err
	}

	if state != nil {
		return state, nil
	}

	// Cache miss - fetch from Polar (blocking, but rare)
	return u.refreshStateFromPolar(ctx, userID)
}

// refreshStateFromPolar fetches fresh state from Polar and updates cache.
func (u *usageInterceptor) refreshStateFromPolar(ctx context.Context, userID string) (*redisrepo.CachedCustomerState, error) {
	if u.cfg.PolarClient == nil {
		// No Polar client - return default free state
		state := &redisrepo.CachedCustomerState{
			IsPaid:       false,
			PlanType:     "free",
			PolarBalance: 100, // Default daily balance
		}
		return state, nil
	}

	polarState, err := u.cfg.PolarClient.GetCustomerStateByExternalID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Polar state: %w", err)
	}

	state := &redisrepo.CachedCustomerState{
		IsPaid:       polarState.IsPaid,
		PlanType:     polarState.PlanType,
		PolarBalance: polarState.CreditBalance,
		ProductID:    polarState.ProductID,
	}

	// Cache the state and reset consumed counter atomically
	if err := u.cfg.CreditCache.SetStateAndResetConsumed(ctx, userID, state); err != nil {
		log.Warn().Err(err).Str("user_id", userID).Msg("Failed to cache customer state")
	}

	return state, nil
}

// postSendPaid handles post-send actions for paid users.
func (u *usageInterceptor) postSendPaid(ctx context.Context, userID string, count int64) {
	// Ingest event to Polar for metered billing
	if u.cfg.PolarClient != nil {
		if err := u.cfg.PolarClient.IngestEmailEvent(ctx, userID, count); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to ingest Polar event")
		}
	}

	// Trigger debounced state refresh
	u.maybeRefreshAsync(userID)
}

// postSendFree handles post-send actions for free users.
func (u *usageInterceptor) postSendFree(ctx context.Context, userID string, count int64) {
	// Increment consumed counter
	if err := u.cfg.CreditCache.IncrConsumed(ctx, userID, count); err != nil {
		log.Warn().Err(err).Str("user_id", userID).Msg("Failed to increment consumed count")
	}

	// Ingest event to Polar
	if u.cfg.PolarClient != nil {
		if err := u.cfg.PolarClient.IngestEmailEvent(ctx, userID, count); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to ingest Polar event")
		}
	}

	// Trigger debounced state refresh
	u.maybeRefreshAsync(userID)
}

// maybeRefreshAsync triggers async refresh if debounce period has passed.
func (u *usageInterceptor) maybeRefreshAsync(userID string) {
	const debouncePeriod = 30 * time.Second

	u.refreshMu.Lock()
	lastReq, exists := u.lastRefreshReq[userID]
	now := time.Now()
	if exists && now.Sub(lastReq) < debouncePeriod {
		u.refreshMu.Unlock()
		return
	}
	u.lastRefreshReq[userID] = now
	u.refreshMu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := u.refreshStateFromPolar(ctx, userID); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed async state refresh")
		}
	}()
}

// creditExhaustedError creates a credit exhausted error for free users.
func creditExhaustedError(polarBalance, consumed int64) error {
	err := connect.NewError(
		connect.CodeResourceExhausted,
		fmt.Errorf("daily email credits exhausted (%d/%d). Upgrade to a paid plan for more emails.",
			consumed, polarBalance),
	)
	err.Meta().Set("X-Plan", "free")
	err.Meta().Set("X-Credits-Balance", fmt.Sprintf("%d", polarBalance))
	err.Meta().Set("X-Credits-Consumed", fmt.Sprintf("%d", consumed))
	err.Meta().Set("X-Upgrade-URL", "https://simpleemailapi.dev/pricing")
	return err
}

// WrapStreamingClient implements connect.Interceptor (no-op for server).
func (u *usageInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor for server streaming calls.
// Streaming endpoints don't count against email limits.
func (u *usageInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
