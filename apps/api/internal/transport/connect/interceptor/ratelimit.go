package interceptor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	"github.com/emailapi/api/internal/domain"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	transporterrors "github.com/emailapi/api/internal/transport/errors"
)

// RateLimitConfig holds configuration for the rate limit interceptor.
type RateLimitConfig struct {
	RateLimiter          RateLimiterInterface
	RequestsPerMinute    int
	MaxConcurrentStreams int
	Enabled              bool
}

// rateLimitedProcedures defines which procedures are rate limited.
// These are all public APIs that can be called with API keys.
var rateLimitedProcedures = map[string]bool{
	"/v1.EmailService/SendEmail":     true,
	"/v1.EmailService/StreamEvents":  true,
	"/v1.DomainService/AddDomain":    true,
	"/v1.DomainService/GetDomain":    true,
	"/v1.DomainService/ListDomains":  true,
	"/v1.DomainService/VerifyDomain": true,
	"/v1.DomainService/DeleteDomain": true,
}

// streamingProcedures defines which procedures are streaming and need concurrent limit tracking.
var streamingProcedures = map[string]bool{
	"/v1.EmailService/StreamEvents": true,
}

// rateLimitInterceptor implements connect.Interceptor for rate limiting.
type rateLimitInterceptor struct {
	cfg RateLimitConfig
}

// NewRateLimitInterceptor creates a new rate limit interceptor.
// Must be placed AFTER auth interceptor in the chain (needs user ID from context).
func NewRateLimitInterceptor(cfg RateLimitConfig) connect.Interceptor {
	return &rateLimitInterceptor{cfg: cfg}
}

// WrapUnary implements connect.Interceptor for unary calls.
func (r *rateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if !r.cfg.Enabled {
			return next(ctx, req)
		}

		procedure := req.Spec().Procedure

		// Skip non-rate-limited procedures
		if !rateLimitedProcedures[procedure] {
			return next(ctx, req)
		}

		// Get user ID from context (set by auth interceptor)
		userID := GetUserID(ctx)
		if userID == "" {
			// No user ID means no rate limiting (let auth interceptor handle this)
			return next(ctx, req)
		}

		// Check rate limit
		result, err := r.cfg.RateLimiter.Check(
			ctx,
			redisrepo.RateLimitKey(userID),
			r.cfg.RequestsPerMinute,
			time.Minute,
		)
		if err != nil {
			// Log error but allow request (fail open for availability)
			log.Warn().Err(err).Str("user_id", userID).Msg("Rate limit check failed, allowing request")
			return next(ctx, req)
		}

		// If rate limited, return 429 BEFORE calling the handler
		if !result.Allowed {
			return nil, rateLimitError(result, r.cfg.RequestsPerMinute)
		}

		// Rate limit passed, call the handler
		resp, respErr := next(ctx, req)

		// Add rate limit headers to successful responses only
		if resp != nil && respErr == nil {
			setRateLimitHeaders(resp.Header(), r.cfg.RequestsPerMinute, result)
		}

		return resp, respErr
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for server).
func (r *rateLimitInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor for server streaming calls.
func (r *rateLimitInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if !r.cfg.Enabled {
			return next(ctx, conn)
		}

		procedure := conn.Spec().Procedure

		// Skip non-rate-limited procedures
		if !rateLimitedProcedures[procedure] {
			return next(ctx, conn)
		}

		// Get user ID from context (set by auth interceptor)
		userID := GetUserID(ctx)
		if userID == "" {
			return next(ctx, conn)
		}

		// Check request rate limit (stream initiation counts as 1 request)
		result, err := r.cfg.RateLimiter.Check(
			ctx,
			redisrepo.RateLimitKey(userID),
			r.cfg.RequestsPerMinute,
			time.Minute,
		)
		if err != nil {
			log.Warn().Err(err).Str("user_id", userID).Msg("Rate limit check failed, allowing stream")
			// Continue to concurrent stream check
		} else if !result.Allowed {
			return rateLimitError(result, r.cfg.RequestsPerMinute)
		}

		// For streaming procedures, also check concurrent stream limit
		if streamingProcedures[procedure] {
			allowed, err := r.cfg.RateLimiter.IncrementStreams(ctx, userID, r.cfg.MaxConcurrentStreams)
			if err != nil {
				log.Warn().Err(err).Str("user_id", userID).Msg("Stream limit check failed, allowing stream")
			} else if !allowed {
				return transporterrors.ToConnectError(
					domain.ErrMaxConcurrentStreams.Clone().
						WithMeta("max_streams", fmt.Sprintf("%d", r.cfg.MaxConcurrentStreams)),
				)
			}

			// Decrement stream count when connection closes
			defer func() {
				if err := r.cfg.RateLimiter.DecrementStreams(ctx, userID); err != nil {
					log.Warn().Err(err).Str("user_id", userID).Msg("Failed to decrement stream count")
				}
			}()
		}

		// Set rate limit headers on response
		if result != nil {
			setRateLimitHeaders(conn.ResponseHeader(), r.cfg.RequestsPerMinute, result)
		}

		return next(ctx, conn)
	}
}

// setRateLimitHeaders sets standard rate limit headers on the response.
// Follows draft-ietf-httpapi-ratelimit-headers conventions.
func setRateLimitHeaders(headers interface{ Set(key, value string) }, limit int, result *redisrepo.RateLimitResult) {
	headers.Set("RateLimit-Limit", strconv.Itoa(limit))
	headers.Set("RateLimit-Remaining", strconv.Itoa(result.Remaining))
	headers.Set("RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
}

// rateLimitError creates a rate limit exceeded error with retry-after info.
func rateLimitError(result *redisrepo.RateLimitResult, limit int) error {
	retryAfter := time.Until(result.ResetAt).Seconds()
	if retryAfter < 1 {
		retryAfter = 1
	}

	err := domain.ErrRateLimited.Clone().
		WithMeta("limit", strconv.Itoa(limit)).
		WithMeta("remaining", "0").
		WithMeta("reset_at", strconv.FormatInt(result.ResetAt.Unix(), 10)).
		WithMeta("retry_after", strconv.Itoa(int(retryAfter)))

	connectErr := transporterrors.ToConnectError(err)
	// Add headers for compatibility
	connectErr.Meta().Set("RateLimit-Limit", strconv.Itoa(limit))
	connectErr.Meta().Set("RateLimit-Remaining", "0")
	connectErr.Meta().Set("RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
	connectErr.Meta().Set("Retry-After", strconv.Itoa(int(retryAfter)))

	return connectErr
}

// IsRateLimitError checks if an error is a rate limit error.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr.Code() == connect.CodeResourceExhausted &&
			strings.Contains(connectErr.Message(), "rate limit")
	}
	return false
}
