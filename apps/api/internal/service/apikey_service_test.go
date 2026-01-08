package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/domain"
	repoMocks "github.com/emailapi/api/internal/repository/postgres/mocks"
	redisMocks "github.com/emailapi/api/internal/repository/redis/mocks"
	tinybirdMocks "github.com/emailapi/api/internal/repository/tinybird/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
)

func TestAPIKeyService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)
	mockAnalytics := serviceMocks.NewMockAnalytics(ctrl)

	mockUserRepo := repoMocks.NewMockUserRepository(ctrl)
	mockAPIKeyRepo := repoMocks.NewMockAPIKeyRepository(ctrl)
	mockAPIKeyCache := redisMocks.NewMockAPIKeyCacheInterface(ctrl)
	mockActivityRepo := tinybirdMocks.NewMockActivityRepositoryInterface(ctrl)

	// Setup Store mocks
	mockStore.EXPECT().Users().Return(mockUserRepo).AnyTimes()
	mockStore.EXPECT().APIKeys().Return(mockAPIKeyRepo).AnyTimes()

	// Setup Cache/Analytics mocks
	mockCache.EXPECT().APIKey().Return(mockAPIKeyCache).AnyTimes()
	mockAnalytics.EXPECT().Activity().Return(mockActivityRepo).AnyTimes()

	svc := NewAPIKeyService(mockStore, mockCache, mockAnalytics, "hmac-secret")
	ctx := context.Background()
	userID := "user_1"

	t.Run("success", func(t *testing.T) {
		req := &domain.CreateAPIKeyRequest{
			Name: "Test Key",
		}

		mockUserRepo.EXPECT().
			GetByID(ctx, userID).
			Return(&domain.User{ID: userID}, nil)

		mockAPIKeyRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, key *domain.APIKey) error {
				assert.Equal(t, userID, key.UserID)
				assert.Equal(t, req.Name, key.Name)
				assert.NotEmpty(t, key.KeyHash)
				assert.NotEmpty(t, key.KeyPrefix)
				return nil
			})

		mockAPIKeyCache.EXPECT().
			InvalidateByUserID(ctx, userID).
			Return(nil)

		mockActivityRepo.EXPECT().
			LogAPIKey(ctx, userID, gomock.Any(), "create", "success", gomock.Any()).
			Return(nil)

		key, rawKey, err := svc.Create(ctx, userID, req)
		require.NoError(t, err)
		assert.NotEmpty(t, key.ID)
		assert.NotEmpty(t, rawKey)
		assert.Contains(t, rawKey, "sea_live_")
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo.EXPECT().
			GetByID(ctx, userID).
			Return(nil, errors.New("not found"))

		_, _, err := svc.Create(ctx, userID, &domain.CreateAPIKeyRequest{})
		assert.Error(t, err)
	})
}

func TestAPIKeyService_ValidateAndGetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)

	mockUserRepo := repoMocks.NewMockUserRepository(ctrl)
	mockAPIKeyRepo := repoMocks.NewMockAPIKeyRepository(ctrl)
	mockAPIKeyCache := redisMocks.NewMockAPIKeyCacheInterface(ctrl)

	mockStore.EXPECT().Users().Return(mockUserRepo).AnyTimes()
	mockStore.EXPECT().APIKeys().Return(mockAPIKeyRepo).AnyTimes()
	mockCache.EXPECT().APIKey().Return(mockAPIKeyCache).AnyTimes()

	svc := NewAPIKeyService(mockStore, mockCache, nil, "hmac-secret")
	ctx := context.Background()
	userID := "user_1"

	// We need a valid raw key for checksum validation tests.
	// We use the service itself to generate one, mocking the dependencies for this setup call.
	mockUserRepo.EXPECT().GetByID(ctx, userID).Return(&domain.User{ID: userID}, nil).Times(1)
	mockAPIKeyRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil).Times(1)
	mockAPIKeyCache.EXPECT().InvalidateByUserID(ctx, userID).Return(nil).Times(1)

	_, validRawKey, _ := svc.Create(ctx, userID, &domain.CreateAPIKeyRequest{Name: "valid"})

	// Now test ValidateAndGetUser

	t.Run("invalid format", func(t *testing.T) {
		_, _, err := svc.ValidateAndGetUser(ctx, "invalid-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})

	t.Run("cache hit success", func(t *testing.T) {
		mockAPIKeyCache.EXPECT().
			GetByKeyHash(ctx, gomock.Any()).
			Return(&domain.APIKey{
				ID: userID, UserID: userID, IsActive: true,
			}, nil)

		mockUserRepo.EXPECT().
			GetByID(ctx, userID).
			Return(&domain.User{ID: userID, IsActive: true}, nil)

		mockAPIKeyRepo.EXPECT().
			UpdateLastUsed(gomock.Any(), gomock.Any()).
			Return(nil).AnyTimes() // Async

		user, key, err := svc.ValidateAndGetUser(ctx, validRawKey)
		require.NoError(t, err)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, userID, key.ID)
	})

	t.Run("cache miss success", func(t *testing.T) {
		mockAPIKeyCache.EXPECT().
			GetByKeyHash(ctx, gomock.Any()).
			Return(nil, errors.New("miss"))

		mockAPIKeyRepo.EXPECT().
			GetByHash(ctx, gomock.Any()).
			Return(&domain.APIKey{
				ID: userID, UserID: userID, IsActive: true,
			}, nil)

		mockAPIKeyCache.EXPECT().
			SetByKeyHash(ctx, gomock.Any(), gomock.Any()).
			Return(nil)

		mockUserRepo.EXPECT().
			GetByID(ctx, userID).
			Return(&domain.User{ID: userID, IsActive: true}, nil)

		mockAPIKeyRepo.EXPECT().
			UpdateLastUsed(gomock.Any(), gomock.Any()).
			Return(nil).AnyTimes()

		user, key, err := svc.ValidateAndGetUser(ctx, validRawKey)
		require.NoError(t, err)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, userID, key.UserID)
	})
}
