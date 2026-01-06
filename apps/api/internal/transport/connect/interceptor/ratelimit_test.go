package interceptor

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	redisrepo "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/transport/connect/interceptor/mocks"
)

func TestRateLimitInterceptor_WrapUnary(t *testing.T) {
	t.Run("disabled skips limiting", func(t *testing.T) {
		interceptor := NewRateLimitInterceptor(RateLimitConfig{Enabled: false})
		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)
		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("non-limited procedure skips", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		interceptor := NewRateLimitInterceptor(RateLimitConfig{Enabled: true})
		// Assuming GetPlans is not in rateLimitedProcedures map (it's public)
		req := newMockUnaryRequest("/v1.BillingService/GetPlans", nil)

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("no user ID skips", func(t *testing.T) {
		interceptor := NewRateLimitInterceptor(RateLimitConfig{Enabled: true})
		// Context without user ID
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("under limit allows request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLimiter := mocks.NewMockRateLimiterInterface(ctrl)
		interceptor := NewRateLimitInterceptor(RateLimitConfig{
			Enabled:           true,
			RateLimiter:       mockLimiter,
			RequestsPerMinute: 100,
		})

		ctx := newTestContext("user_1")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		mockLimiter.EXPECT().
			Check(ctx, redisrepo.RateLimitKey("user_1"), 100, time.Minute).
			Return(&redisrepo.RateLimitResult{
				Allowed:   true,
				Remaining: 99,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil)

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "100", resp.Header().Get("RateLimit-Limit"))
		assert.Equal(t, "99", resp.Header().Get("RateLimit-Remaining"))
	})

	t.Run("over limit blocks request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLimiter := mocks.NewMockRateLimiterInterface(ctrl)
		interceptor := NewRateLimitInterceptor(RateLimitConfig{
			Enabled:           true,
			RateLimiter:       mockLimiter,
			RequestsPerMinute: 100,
		})

		ctx := newTestContext("user_1")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		resetAt := time.Now().Add(30 * time.Second)
		mockLimiter.EXPECT().
			Check(ctx, redisrepo.RateLimitKey("user_1"), 100, time.Minute).
			Return(&redisrepo.RateLimitResult{
				Allowed:   false,
				Remaining: 0,
				ResetAt:   resetAt,
			}, nil)

		_, err := interceptor.WrapUnary(nil)(ctx, req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

		// Verify error metadata
		var connectErr *connect.Error
		errors.As(err, &connectErr)
		assert.Equal(t, "100", connectErr.Meta().Get("RateLimit-Limit"))
		assert.Equal(t, "0", connectErr.Meta().Get("RateLimit-Remaining"))
	})

	t.Run("redis error fails open", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLimiter := mocks.NewMockRateLimiterInterface(ctrl)
		interceptor := NewRateLimitInterceptor(RateLimitConfig{
			Enabled:           true,
			RateLimiter:       mockLimiter,
			RequestsPerMinute: 100,
		})

		ctx := newTestContext("user_1")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		mockLimiter.EXPECT().
			Check(ctx, gomock.Any(), 100, time.Minute).
			Return(nil, errors.New("redis error"))

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		// Should succeed despite Redis error
		resp, err := interceptor.WrapUnary(mockHandler)(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
	t.Run("zero limit blocks everything", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLimiter := mocks.NewMockRateLimiterInterface(ctrl)
		interceptor := NewRateLimitInterceptor(RateLimitConfig{
			Enabled:           true,
			RateLimiter:       mockLimiter,
			RequestsPerMinute: 0, // Block everything
		})

		ctx := newTestContext("user_zero")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		mockLimiter.EXPECT().
			Check(ctx, gomock.Any(), 0, time.Minute).
			Return(&redisrepo.RateLimitResult{
				Allowed:   false,
				Remaining: 0,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil)

		_, err := interceptor.WrapUnary(nil)(ctx, req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	})
}

func TestRateLimitInterceptor_WrapStreaming(t *testing.T) {
	t.Run("concurrent streams limit zero blocks", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockLimiter := mocks.NewMockRateLimiterInterface(ctrl)
		interceptor := NewRateLimitInterceptor(RateLimitConfig{
			Enabled:              true,
			RequestsPerMinute:    100, // Non-zero to pass unary check
			MaxConcurrentStreams: 0,   // Block all streams
			RateLimiter:          mockLimiter,
		})

		ctx := newTestContext("user_streams")
		conn := newMockStreamingConn("/v1.EmailService/StreamEvents", nil)

		// Mock handler
		mockHandler := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
			return nil
		}

		// 1. First the unary rate limit check happens
		mockLimiter.EXPECT().
			Check(ctx, gomock.Any(), 100, time.Minute).
			Return(&redisrepo.RateLimitResult{
				Allowed:   true,
				Remaining: 50,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil)

		// 2. Then concurrent stream check
		mockLimiter.EXPECT().
			IncrementStreams(ctx, "user_streams", 0).
			Return(false, nil) // Rejected

		err := interceptor.WrapStreamingHandler(mockHandler)(ctx, conn)
		require.Error(t, err)
		assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		assert.Contains(t, err.Error(), "too many concurrent streams")
	})
}
