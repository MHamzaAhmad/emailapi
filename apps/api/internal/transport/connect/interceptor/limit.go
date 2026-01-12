package interceptor

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/limit"
	transporterrors "github.com/emailapi/api/internal/transport/errors"
)

// LimitConfig holds configuration for the limit interceptor.
type LimitConfig struct {
	// Engine is the pre-configured limit engine
	Engine *limit.Engine

	// Feature flag
	Enabled bool
}

// limitProcedures defines which procedures count against limits.
var limitProcedures = map[string]bool{
	"/v1.EmailService/SendEmail": true,
}

type limitInterceptor struct {
	cfg LimitConfig
}

// NewLimitInterceptor creates a limit interceptor.
// Uses limit.Engine for parallel checking.
// Blocks on hard suspend or limits exceeded.
// Usage increment happens in worker after successful send.
func NewLimitInterceptor(cfg LimitConfig) connect.Interceptor {
	return &limitInterceptor{cfg: cfg}
}

// WrapUnary implements connect.Interceptor for unary calls.
// Blocks requests that exceed limits. Usage tracking happens in worker.
func (l *limitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if !l.cfg.Enabled || l.cfg.Engine == nil {
			return next(ctx, req)
		}

		procedure := req.Spec().Procedure
		if !limitProcedures[procedure] {
			return next(ctx, req)
		}

		userID := GetUserID(ctx)
		if userID == "" {
			return next(ctx, req)
		}

		// Run all limit checks in parallel via engine
		result, err := l.cfg.Engine.Check(ctx, userID)
		if err != nil {
			// Fail open on error
			log.Warn().Err(err).Str("user_id", userID).Msg("Limit check failed, allowing request")
			return next(ctx, req)
		}

		// Block if not allowed (hard suspend, limits exceeded, etc.)
		if !result.Allowed {
			log.Warn().
				Str("user_id", userID).
				Str("reason", string(result.Reason)).
				Interface("meta", result.Meta).
				Msg("Request blocked by limit check")
			return nil, limitResultToError(result)
		}

		// Allowed - proceed with request
		// Usage tracking happens in worker after successful send
		return next(ctx, req)
	}
}

// limitResultToError converts a limit.Result to Connect error.
func limitResultToError(result *limit.Result) error {
	switch result.Reason {
	case limit.ReasonHardSuspended:
		return transporterrors.ToConnectError(domain.ErrAccountSuspended)

	case limit.ReasonDailyExceeded:
		dailyLimit := result.Meta["daily_limit"]
		dailyUsage := result.Meta["daily_usage"]
		err := domain.ErrDailyLimitExceeded.Clone().
			WithMeta("daily_limit", dailyLimit).
			WithMeta("daily_usage", dailyUsage).
			WithMeta("upgrade_url", "https://simpleemailapi.dev/pricing")
		connectErr := transporterrors.ToConnectError(err)
		connectErr.Meta().Set("X-Daily-Limit", dailyLimit)
		connectErr.Meta().Set("X-Daily-Usage", dailyUsage)
		return connectErr

	case limit.ReasonCreditsExhausted:
		err := domain.ErrCreditsExhausted.Clone().
			WithMeta("plan", "free").
			WithMeta("upgrade_url", "https://simpleemailapi.dev/pricing")
		return transporterrors.ToConnectError(err)

	default:
		return transporterrors.ToConnectError(
			domain.ErrInternal.Clone().WithMeta("reason", fmt.Sprintf("limit: %s", result.Reason)),
		)
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for server).
func (l *limitInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor for server streaming calls.
func (l *limitInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
