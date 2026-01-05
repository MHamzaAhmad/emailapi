package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/domain"
	polarMocks "github.com/emailapi/api/internal/external/polar/mocks"
	repoMocks "github.com/emailapi/api/internal/repository/postgres/mocks"
	redisMocks "github.com/emailapi/api/internal/repository/redis/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
)

func init() {
	// Initialize plan mappings for tests
	domain.InitProductMappings("prod_starter", "prod_growth")
}

func TestBillingService_GetPlans(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := NewBillingService(nil, nil, nil)

	plans := svc.GetPlans()

	require.Len(t, plans, 3)
	assert.Equal(t, "free", plans[0].ID)
	assert.Equal(t, "starter", plans[1].ID)
	assert.Equal(t, "growth", plans[2].ID)

	// Verify limits are populated
	assert.Equal(t, int64(3000), plans[0].MonthlyLimit)
	assert.Equal(t, int64(100), plans[0].DailyLimit)
	assert.Equal(t, int64(50000), plans[1].MonthlyLimit)
	assert.Equal(t, int64(-1), plans[1].DailyLimit)
}

func TestBillingService_SyncSubscription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockUserRepo := repoMocks.NewMockUserRepository(ctrl)
	mockPolar := polarMocks.NewMockClient(ctrl)
	mockUsageCache := redisMocks.NewMockUsageCacheInterface(ctrl)

	mockStore.EXPECT().Users().Return(mockUserRepo).AnyTimes()

	svc := NewBillingService(mockPolar, mockUsageCache, mockStore)
	ctx := context.Background()
	userID := "user_123"

	t.Run("no polar client", func(t *testing.T) {
		svcNoPolar := NewBillingService(nil, nil, mockStore)
		user := &domain.User{ID: userID, Plan: domain.UserPlanFree}
		mockUserRepo.EXPECT().GetByID(ctx, userID).Return(user, nil)

		result, err := svcNoPolar.SyncSubscription(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, userID, result.ID)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByID(ctx, userID).Return(nil, errors.New("not found"))

		_, err := svc.SyncSubscription(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("no polar customer", func(t *testing.T) {
		user := &domain.User{ID: userID, Plan: domain.UserPlanFree}
		mockUserRepo.EXPECT().GetByID(ctx, userID).Return(user, nil)
		mockPolar.EXPECT().GetCustomerByExternalID(ctx, userID).Return(nil, errors.New("not found"))

		result, err := svc.SyncSubscription(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, domain.UserPlanFree, result.Plan)
	})
}

func TestBillingService_CreateCheckoutSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockPolar := polarMocks.NewMockClient(ctrl)

	svc := NewBillingService(mockPolar, nil, mockStore)
	ctx := context.Background()
	userID := "user_123"
	successURL := "https://app.com/success"

	t.Run("no polar client", func(t *testing.T) {
		svcNoPolar := NewBillingService(nil, nil, nil)
		_, err := svcNoPolar.CreateCheckoutSession(ctx, userID, "starter", successURL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "polar not configured")
	})

	t.Run("invalid plan", func(t *testing.T) {
		_, err := svc.CreateCheckoutSession(ctx, userID, "invalid", successURL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plan")
	})

	t.Run("success", func(t *testing.T) {
		checkoutURL := "https://polar.sh/checkout/abc"
		mockPolar.EXPECT().
			CreateCheckoutSession(ctx, gomock.Any()).
			Return(checkoutURL, nil)

		result, err := svc.CreateCheckoutSession(ctx, userID, "starter", successURL)
		require.NoError(t, err)
		assert.Equal(t, checkoutURL, result)
	})
}

func TestBillingService_GetCustomerPortalUrl(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockUserRepo := repoMocks.NewMockUserRepository(ctrl)
	mockPolar := polarMocks.NewMockClient(ctrl)

	mockStore.EXPECT().Users().Return(mockUserRepo).AnyTimes()

	svc := NewBillingService(mockPolar, nil, mockStore)
	ctx := context.Background()
	userID := "user_123"
	polarCustomerID := "cus_polar_abc"

	t.Run("no polar client", func(t *testing.T) {
		svcNoPolar := NewBillingService(nil, nil, nil)
		_, err := svcNoPolar.GetCustomerPortalUrl(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "polar not configured")
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByID(ctx, userID).Return(nil, errors.New("not found"))
		_, err := svc.GetCustomerPortalUrl(ctx, userID)
		assert.Error(t, err)
	})

	t.Run("no polar customer id", func(t *testing.T) {
		user := &domain.User{ID: userID, PolarCustomerID: nil}
		mockUserRepo.EXPECT().GetByID(ctx, userID).Return(user, nil)
		_, err := svc.GetCustomerPortalUrl(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no Polar customer ID")
	})

	t.Run("success", func(t *testing.T) {
		user := &domain.User{ID: userID, PolarCustomerID: &polarCustomerID}
		portalURL := "https://polar.sh/portal/abc"
		mockUserRepo.EXPECT().GetByID(ctx, userID).Return(user, nil)
		mockPolar.EXPECT().CreateCustomerPortal(ctx, polarCustomerID).Return(portalURL, nil)

		result, err := svc.GetCustomerPortalUrl(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, portalURL, result)
	})
}

func TestDomain_PlanFunctions(t *testing.T) {
	t.Run("GetPlanFromProductID", func(t *testing.T) {
		assert.Equal(t, domain.UserPlanStarter, domain.GetPlanFromProductID("prod_starter"))
		assert.Equal(t, domain.UserPlanGrowth, domain.GetPlanFromProductID("prod_growth"))
		assert.Equal(t, domain.UserPlanFree, domain.GetPlanFromProductID("unknown"))
	})

	t.Run("GetProductIDFromPlanID", func(t *testing.T) {
		productID, ok := domain.GetProductIDFromPlanID("starter")
		assert.True(t, ok)
		assert.Equal(t, "prod_starter", productID)

		productID, ok = domain.GetProductIDFromPlanID("growth")
		assert.True(t, ok)
		assert.Equal(t, "prod_growth", productID)

		_, ok = domain.GetProductIDFromPlanID("free")
		assert.False(t, ok)
	})

	t.Run("AllPlans", func(t *testing.T) {
		plans := domain.AllPlans()
		assert.Len(t, plans, 3)
		assert.Equal(t, domain.UserPlanFree, plans[0])
		assert.Equal(t, domain.UserPlanStarter, plans[1])
		assert.Equal(t, domain.UserPlanGrowth, plans[2])
	})
}
