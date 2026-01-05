package interceptor

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/polar"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// UsageConfig holds configuration for the usage limit interceptor.
type UsageConfig struct {
	UsageCache  redisrepo.UsageCacheInterface
	UserFetcher UserFetcher // Interface to get user by ID
	PolarClient polar.Client
	Enabled     bool
}

// UserFetcher defines interface to fetch user details.
type UserFetcher interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// usageProcedures defines which procedures count against usage limits.
var usageProcedures = map[string]bool{
	"/v1.EmailService/SendEmail": true,
}

type usageInterceptor struct {
	cfg UsageConfig
}

// NewUsageInterceptor creates a new usage limit interceptor.
// Must be placed AFTER auth interceptor (needs user ID from context).
// Must be placed BEFORE rate limit interceptor (usage = billing, rate = abuse prevention).
func NewUsageInterceptor(cfg UsageConfig) connect.Interceptor {
	return &usageInterceptor{cfg: cfg}
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

		// Fast path: check cached plan state
		state, err := u.cfg.UsageCache.GetPlanState(ctx, userID)
		if err != nil {
			// Redis error - log and allow (fail open)
			log.Warn().Err(err).Str("user_id", userID).Msg("Usage cache error, allowing request")
			return next(ctx, req)
		}

		if state == nil {
			// Cache miss - hydrate from DB
			state, err = u.hydrateState(ctx, userID)
			if err != nil {
				log.Warn().Err(err).Str("user_id", userID).Msg("Failed to hydrate plan state, allowing request")
				return next(ctx, req)
			}
		}

		// Check limits (count 1 email per request - can be enhanced to count recipients)
		emailCount := int64(1)

		// Check daily limit (Free plan only)
		if !state.IsWithinDailyLimit(emailCount) && state.HardLimit {
			return nil, dailyLimitError(state)
		}

		// Check monthly limit
		if !state.IsWithinMonthlyLimit(emailCount) && state.HardLimit {
			return nil, monthlyLimitError(state)
		}

		// Proceed with request
		resp, respErr := next(ctx, req)

		// If successful, increment counters and optionally ingest to Polar (async)
		if respErr == nil && resp != nil {
			go u.postSendActions(context.Background(), userID, emailCount, state)
		}

		return resp, respErr
	}
}

// hydrateState loads user plan and usage from DB/cache.
func (u *usageInterceptor) hydrateState(ctx context.Context, userID string) (*redisrepo.PlanState, error) {
	// Get user from DB for plan
	user, err := u.cfg.UserFetcher.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get usage from Redis counters
	dailyUsage, _ := u.cfg.UsageCache.GetDailyUsage(ctx, userID)
	monthlyUsage, _ := u.cfg.UsageCache.GetMonthlyUsage(ctx, userID)

	// Get plan config
	planConfig := user.Plan.GetConfig()

	state := &redisrepo.PlanState{
		Plan:         string(user.Plan),
		MonthlyLimit: planConfig.MonthlyLimit,
		DailyLimit:   planConfig.DailyLimit,
		HardLimit:    planConfig.HardLimit,
		BillAllUsage: planConfig.BillAllUsage,
		MonthlyUsage: monthlyUsage,
		DailyUsage:   dailyUsage,
		CachedAt:     time.Now().Unix(),
	}

	// Cache the state
	if err := u.cfg.UsageCache.SetPlanState(ctx, userID, state); err != nil {
		log.Warn().Err(err).Str("user_id", userID).Msg("Failed to cache plan state")
	}

	return state, nil
}

// postSendActions runs after a successful send.
func (u *usageInterceptor) postSendActions(ctx context.Context, userID string, count int64, state *redisrepo.PlanState) {
	// 1. Increment Redis counters
	if err := u.cfg.UsageCache.IncrementUsage(ctx, userID, count); err != nil {
		log.Warn().Err(err).Str("user_id", userID).Msg("Failed to increment usage")
	}

	// 2. Ingest to Polar if needed
	if u.cfg.PolarClient == nil {
		return
	}

	needsPolar := state.BillAllUsage // PAYG: all emails
	if !needsPolar && state.MonthlyLimit > 0 {
		// Scale: check if this is overage
		needsPolar = state.MonthlyUsage+count > state.MonthlyLimit
	}

	if needsPolar {
		if err := u.cfg.PolarClient.IngestEmailEvent(ctx, userID, count); err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Failed to ingest Polar event")
		}
	}
}

// dailyLimitError creates a daily limit exceeded error.
func dailyLimitError(state *redisrepo.PlanState) error {
	err := connect.NewError(
		connect.CodeResourceExhausted,
		fmt.Errorf("daily email limit exceeded (%d/%d). Upgrade to Scale or PAYG plan for no daily limits.",
			state.DailyUsage, state.DailyLimit),
	)
	err.Meta().Set("X-Plan", state.Plan)
	err.Meta().Set("X-Daily-Limit", fmt.Sprintf("%d", state.DailyLimit))
	err.Meta().Set("X-Daily-Usage", fmt.Sprintf("%d", state.DailyUsage))
	err.Meta().Set("X-Upgrade-URL", "https://simpleemailapi.dev/pricing")
	return err
}

// monthlyLimitError creates a monthly limit exceeded error.
func monthlyLimitError(state *redisrepo.PlanState) error {
	err := connect.NewError(
		connect.CodeResourceExhausted,
		fmt.Errorf("monthly email limit exceeded (%d/%d). Please upgrade your plan.",
			state.MonthlyUsage, state.MonthlyLimit),
	)
	err.Meta().Set("X-Plan", state.Plan)
	err.Meta().Set("X-Monthly-Limit", fmt.Sprintf("%d", state.MonthlyLimit))
	err.Meta().Set("X-Monthly-Usage", fmt.Sprintf("%d", state.MonthlyUsage))
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
