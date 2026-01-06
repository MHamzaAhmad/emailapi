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

	"github.com/emailapi/api/internal/external/polar"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/transport/connect/interceptor/mocks"
)

func TestUsageInterceptor_WrapUnary(t *testing.T) {
	t.Run("disabled skips usage check", func(t *testing.T) {
		interceptor := NewUsageInterceptor(UsageConfig{Enabled: false})
		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)
		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("non-usage procedure skips", func(t *testing.T) {
		interceptor := NewUsageInterceptor(UsageConfig{Enabled: true})
		// Domain service doesn't consume credits
		req := newMockUnaryRequest("/v1.DomainService/AddDomain", nil)

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("paid user allowed immediately", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCreditCache := mocks.NewMockCreditCacheInterface(ctrl)
		mockPolar := mocks.NewMockPolarClientInterface(ctrl)

		interceptor := NewUsageInterceptor(UsageConfig{
			Enabled:     true,
			CreditCache: mockCreditCache,
			PolarClient: mockPolar,
		})

		ctx := newTestContext("user_paid")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		// Cache hit - user is paid
		mockCreditCache.EXPECT().
			GetState(ctx, "user_paid").
			Return(&redisrepo.CachedCustomerState{
				IsPaid: true,
			}, nil)

		// Async Polar ingestion (fire & forget, tricky to test deterministically without sleep)
		// mocking expectation anyway
		mockPolar.EXPECT().
			IngestEmailEvent(gomock.Any(), "user_paid", int64(1)).
			Return(nil).
			AnyTimes()

		// Mock async refresh call
		mockPolar.EXPECT().
			GetCustomerStateByExternalID(gomock.Any(), "user_paid").
			Return(&polar.CustomerState{IsPaid: true}, nil).
			AnyTimes()
		mockCreditCache.EXPECT().
			SetStateAndResetConsumed(gomock.Any(), "user_paid", gomock.Any()).
			Return(nil).
			AnyTimes()

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		time.Sleep(10 * time.Millisecond)
	})

	t.Run("free user under daily limit allowed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCreditCache := mocks.NewMockCreditCacheInterface(ctrl)
		mockPolar := mocks.NewMockPolarClientInterface(ctrl)

		interceptor := NewUsageInterceptor(UsageConfig{
			Enabled:        true,
			CreditCache:    mockCreditCache,
			PolarClient:    mockPolar,
			FreeDailyLimit: 100,
		})

		ctx := newTestContext("user_free")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		// 1. Get State
		mockCreditCache.EXPECT().
			GetState(ctx, "user_free").
			Return(&redisrepo.CachedCustomerState{
				IsPaid:       false,
				PolarBalance: 1000,
			}, nil)

		// 2. Check Daily Usage (under limit)
		mockCreditCache.EXPECT().
			GetDailyUsage(ctx, "user_free").
			Return(int64(50), nil)

		// 3. Check Monthly Credits (consumed for today 0, so 1000 > 0)
		mockCreditCache.EXPECT().
			GetConsumed(ctx, "user_free").
			Return(int64(0), nil)

		// 4. Post-Send: Increment counters
		mockCreditCache.EXPECT().
			IncrDailyUsage(gomock.Any(), "user_free", int64(1)).
			Return(nil).AnyTimes()
		mockCreditCache.EXPECT().
			IncrConsumed(gomock.Any(), "user_free", int64(1)).
			Return(nil).AnyTimes()
		mockPolar.EXPECT().
			IngestEmailEvent(gomock.Any(), "user_free", int64(1)).
			Return(nil).AnyTimes()

		// Async refresh
		mockPolar.EXPECT().
			GetCustomerStateByExternalID(gomock.Any(), "user_free").
			Return(&polar.CustomerState{IsPaid: false, CreditBalance: 1000}, nil).
			AnyTimes()
		mockCreditCache.EXPECT().
			SetStateAndResetConsumed(gomock.Any(), "user_free", gomock.Any()).
			Return(nil).AnyTimes()

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		time.Sleep(10 * time.Millisecond)
	})

	t.Run("free user at daily limit blocked", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCreditCache := mocks.NewMockCreditCacheInterface(ctrl)

		interceptor := NewUsageInterceptor(UsageConfig{
			Enabled:        true,
			CreditCache:    mockCreditCache,
			FreeDailyLimit: 100,
		})

		ctx := newTestContext("user_free_limit")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		mockCreditCache.EXPECT().
			GetState(ctx, "user_free_limit").
			Return(&redisrepo.CachedCustomerState{
				IsPaid: false,
			}, nil)

		mockCreditCache.EXPECT().
			GetDailyUsage(ctx, "user_free_limit").
			Return(int64(100), nil)

		_, err := interceptor.WrapUnary(nil)(ctx, req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		assert.Contains(t, err.Error(), "daily email limit exceeded")
	})

	t.Run("free user credits exhausted blocked", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCreditCache := mocks.NewMockCreditCacheInterface(ctrl)

		interceptor := NewUsageInterceptor(UsageConfig{
			Enabled:        true,
			CreditCache:    mockCreditCache,
			FreeDailyLimit: 100,
		})

		ctx := newTestContext("user_no_credits")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		mockCreditCache.EXPECT().
			GetState(ctx, "user_no_credits").
			Return(&redisrepo.CachedCustomerState{
				IsPaid:       false,
				PolarBalance: 50,
			}, nil)

		mockCreditCache.EXPECT().
			GetDailyUsage(ctx, "user_no_credits").
			Return(int64(10), nil)

		mockCreditCache.EXPECT().
			GetConsumed(ctx, "user_no_credits").
			Return(int64(50), nil) // Consumed >= Balance

		_, err := interceptor.WrapUnary(nil)(ctx, req)
		require.Error(t, err)
		assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		assert.Contains(t, err.Error(), "monthly email credits exhausted")
	})

	t.Run("cache miss fetches from polar", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCreditCache := mocks.NewMockCreditCacheInterface(ctrl)
		mockPolar := mocks.NewMockPolarClientInterface(ctrl)

		interceptor := NewUsageInterceptor(UsageConfig{
			Enabled:     true,
			CreditCache: mockCreditCache,
			PolarClient: mockPolar,
		})

		ctx := newTestContext("user_miss")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		// Cache Miss
		mockCreditCache.EXPECT().GetState(ctx, "user_miss").Return(nil, nil)

		// Fetch from Polar
		mockPolar.EXPECT().
			GetCustomerStateByExternalID(ctx, "user_miss").
			Return(&polar.CustomerState{
				IsPaid:        false,
				PlanType:      "free",
				CreditBalance: 500,
			}, nil)

		// Cache Result
		mockCreditCache.EXPECT().
			SetStateAndResetConsumed(ctx, "user_miss", gomock.Any()).
			Return(nil)

		// Continue with checks...
		mockCreditCache.EXPECT().GetDailyUsage(ctx, "user_miss").Return(int64(0), nil)
		mockCreditCache.EXPECT().GetConsumed(ctx, "user_miss").Return(int64(0), nil)
		mockCreditCache.EXPECT().IncrDailyUsage(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockCreditCache.EXPECT().IncrConsumed(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		mockPolar.EXPECT().IngestEmailEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		// Async refresh (happens in background)
		mockPolar.EXPECT().
			GetCustomerStateByExternalID(gomock.Any(), "user_miss").
			Return(&polar.CustomerState{IsPaid: false, CreditBalance: 500}, nil).
			AnyTimes()
		mockCreditCache.EXPECT().
			SetStateAndResetConsumed(gomock.Any(), "user_miss", gomock.Any()).
			Return(nil).
			AnyTimes()

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		time.Sleep(10 * time.Millisecond)
	})

	t.Run("credit cache error fails open", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCreditCache := mocks.NewMockCreditCacheInterface(ctrl)
		interceptor := NewUsageInterceptor(UsageConfig{
			Enabled:     true,
			CreditCache: mockCreditCache,
		})

		ctx := newTestContext("user_fail_open")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		mockCreditCache.EXPECT().
			GetState(ctx, "user_fail_open").
			Return(nil, errors.New("redis error"))

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		// Should succeed despite error
		resp, err := interceptor.WrapUnary(mockHandler)(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("provisioning fails if user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCreditCache := mocks.NewMockCreditCacheInterface(ctrl)
		mockUserRepo := mocks.NewMockUserRepositoryInterface(ctrl)
		mockPolar := mocks.NewMockPolarClientInterface(ctrl)

		interceptor := NewUsageInterceptor(UsageConfig{
			Enabled:       true,
			CreditCache:   mockCreditCache,
			UserRepo:      mockUserRepo,
			PolarClient:   mockPolar,
			FreeProductID: "free_prod_id",
		})

		ctx := newTestContext("user_unknown")
		req := newMockUnaryRequest("/v1.EmailService/SendEmail", nil)

		// 1. Cache miss
		mockCreditCache.EXPECT().GetState(ctx, "user_unknown").Return(nil, nil)

		// 2. Polar lookup returns error (not found in Polar)
		mockPolar.EXPECT().
			GetCustomerStateByExternalID(ctx, "user_unknown").
			Return(nil, errors.New("not found"))

		// 3. User Repo lookup failure (or user not found)
		mockUserRepo.EXPECT().
			GetByID(ctx, "user_unknown").
			Return(nil, errors.New("user not found"))

		// Should fail to provision but fail OPEN to free tier

		// 4. Fallback to free tier checks
		mockCreditCache.EXPECT().
			GetDailyUsage(ctx, "user_unknown").
			Return(int64(0), nil) // Under limit

		mockCreditCache.EXPECT().
			GetConsumed(ctx, "user_unknown").
			Return(int64(0), nil)

		// 5. Increment counters
		mockCreditCache.EXPECT().
			IncrDailyUsage(gomock.Any(), "user_unknown", int64(1)).
			Return(nil).AnyTimes()
		mockCreditCache.EXPECT().
			IncrConsumed(gomock.Any(), "user_unknown", int64(1)).
			Return(nil).AnyTimes()
		mockPolar.EXPECT().
			IngestEmailEvent(gomock.Any(), "user_unknown", int64(1)).
			Return(nil).AnyTimes()

		// Async refresh will happen again?
		// maybeRefreshAsync checks lastRefreshReq.
		// Since we just fetched (failed) in getOrFetchState, it might trigger async if logic dictates.
		// Add expectations for async refresh
		mockPolar.EXPECT().
			GetCustomerStateByExternalID(gomock.Any(), "user_unknown").
			Return(nil, errors.New("not found")).
			AnyTimes()
		mockUserRepo.EXPECT().
			GetByID(gomock.Any(), "user_unknown").
			Return(nil, errors.New("user not found")).
			AnyTimes()
		mockCreditCache.EXPECT().
			SetStateAndResetConsumed(gomock.Any(), "user_unknown", gomock.Any()).
			Return(nil).
			AnyTimes()

		mockHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		}

		resp, err := interceptor.WrapUnary(mockHandler)(ctx, req)
		require.NoError(t, err) // Fails open as free user
		assert.NotNil(t, resp)

		time.Sleep(10 * time.Millisecond)
	})
}
