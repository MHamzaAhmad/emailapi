package interceptor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// UsageConfig holds configuration for the usage limit interceptor.
type UsageConfig struct {
	CreditCache    CreditCacheInterface
	PolarClient    PolarClientInterface
	UserRepo       UserRepositoryInterface // For looking up/updating user
	Enabled        bool
	FreeDailyLimit int64
	FreeProductID  string // Polar product ID for free plan (for re-provisioning)
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

		// Free users: check daily limit first
		dailyUsage, err := u.cfg.CreditCache.GetDailyUsage(ctx, userID)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to get daily usage, allowing request")
			dailyUsage = 0
		}

		dailyLimit := u.cfg.FreeDailyLimit
		if dailyLimit <= 0 {
			dailyLimit = 100 // Default
		}

		if dailyUsage >= dailyLimit {
			return nil, dailyLimitError(dailyUsage, dailyLimit)
		}

		// Free users: check monthly credit balance
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

		// Post-send: increment daily and consumed counters (async)
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
// If user has no Polar customer or subscription, it will attempt to provision them.
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
		// User might not have a Polar customer or subscription - try to provision
		log.Info().Err(err).Str("user_id", userID).Msg("Could not fetch Polar state - attempting to provision")

		if provisionErr := u.ensureCustomerAndSubscription(ctx, userID); provisionErr != nil {
			log.Warn().Err(provisionErr).Str("user_id", userID).Msg("Failed to provision - treating as free user")
			state := &redisrepo.CachedCustomerState{
				IsPaid:       false,
				PlanType:     "free",
				PolarBalance: 100,
			}
			return state, nil
		}

		// Retry fetching state after provisioning
		polarState, err = u.cfg.PolarClient.GetCustomerStateByExternalID(ctx, userID)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Still could not fetch Polar state after provisioning")
			state := &redisrepo.CachedCustomerState{
				IsPaid:       false,
				PlanType:     "free",
				PolarBalance: 100,
			}
			return state, nil
		}
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

// ensureCustomerAndSubscription creates Polar customer and free subscription if missing.
func (u *usageInterceptor) ensureCustomerAndSubscription(ctx context.Context, userID string) error {
	if u.cfg.PolarClient == nil || u.cfg.UserRepo == nil || u.cfg.FreeProductID == "" {
		return fmt.Errorf("polar provisioning not configured")
	}

	// Get user details
	user, err := u.cfg.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	var customerID string

	// Check if user already has a Polar customer ID
	if user.PolarCustomerID != nil && *user.PolarCustomerID != "" {
		customerID = *user.PolarCustomerID
	} else {
		// Try to get existing customer from Polar
		customer, err := u.cfg.PolarClient.GetCustomerByExternalID(ctx, userID)
		if err != nil {
			// Create new customer
			customerID, err = u.cfg.PolarClient.CreateCustomer(ctx, userID, user.Email, user.Name)
			if err != nil {
				return fmt.Errorf("failed to create Polar customer: %w", err)
			}
			log.Info().Str("user_id", userID).Str("customer_id", customerID).Msg("Created Polar customer")
		} else {
			customerID = customer.ID
		}

		// Save customer ID to database
		if err := u.cfg.UserRepo.UpdatePolarCustomerID(ctx, userID, customerID); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to save Polar customer ID")
		}
	}

	// Check if subscription exists
	sub, err := u.cfg.PolarClient.GetActiveSubscriptionByExternalID(ctx, userID)
	if err != nil || sub == nil {
		// Create free subscription
		if err := u.cfg.PolarClient.CreateFreeSubscription(ctx, customerID, u.cfg.FreeProductID); err != nil {
			return fmt.Errorf("failed to create free subscription: %w", err)
		}
		log.Info().Str("user_id", userID).Str("customer_id", customerID).Msg("Created free subscription")
	}

	return nil
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
	// Increment daily usage counter
	if err := u.cfg.CreditCache.IncrDailyUsage(ctx, userID, count); err != nil {
		log.Warn().Err(err).Str("user_id", userID).Msg("Failed to increment daily usage")
	}

	// Increment consumed counter (for monthly balance)
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

// creditExhaustedError creates a credit exhausted error for free users (monthly).
func creditExhaustedError(polarBalance, consumed int64) error {
	err := connect.NewError(
		connect.CodeResourceExhausted,
		fmt.Errorf("monthly email credits exhausted (%d/%d). Upgrade to a paid plan for more emails.",
			consumed, polarBalance),
	)
	err.Meta().Set("X-Plan", "free")
	err.Meta().Set("X-Credits-Balance", fmt.Sprintf("%d", polarBalance))
	err.Meta().Set("X-Credits-Consumed", fmt.Sprintf("%d", consumed))
	err.Meta().Set("X-Upgrade-URL", "https://simpleemailapi.dev/pricing")
	return err
}

// dailyLimitError creates a daily limit exceeded error for free users.
func dailyLimitError(usage, limit int64) error {
	err := connect.NewError(
		connect.CodeResourceExhausted,
		fmt.Errorf("daily email limit exceeded (%d/%d). Limit resets at midnight UTC.",
			usage, limit),
	)
	err.Meta().Set("X-Plan", "free")
	err.Meta().Set("X-Daily-Limit", fmt.Sprintf("%d", limit))
	err.Meta().Set("X-Daily-Usage", fmt.Sprintf("%d", usage))
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
