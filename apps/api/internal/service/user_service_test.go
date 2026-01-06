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
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
)

func TestUserService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)
	mockUserRepo := repoMocks.NewMockUserRepository(ctrl)

	mockStore.EXPECT().Users().Return(mockUserRepo).AnyTimes()

	svc := NewUserService(mockStore, nil, mockCache)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		req := &domain.CreateUserRequest{
			Email: "test@example.com",
			Name:  "Test User",
		}

		mockUserRepo.EXPECT().
			GetByEmail(ctx, req.Email).
			Return(nil, nil)

		mockUserRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, u *domain.User) error {
				assert.Equal(t, req.Email, u.Email)
				assert.Equal(t, req.Name, u.Name)
				assert.NotEmpty(t, u.ID)
				return nil
			})

		user, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, req.Email, user.Email)
	})

	t.Run("email exists", func(t *testing.T) {
		req := &domain.CreateUserRequest{Email: "exist@example.com"}

		mockUserRepo.EXPECT().
			GetByEmail(ctx, req.Email).
			Return(&domain.User{ID: "existing"}, nil)

		_, err := svc.Create(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestUserService_GetByExternalID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := serviceMocks.NewMockStore(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)
	mockUserRepo := repoMocks.NewMockUserRepository(ctrl)
	mockUserCache := redisMocks.NewMockUserCacheInterface(ctrl)

	mockStore.EXPECT().Users().Return(mockUserRepo).AnyTimes()
	mockCache.EXPECT().User().Return(mockUserCache).AnyTimes()

	svc := NewUserService(mockStore, nil, mockCache)
	ctx := context.Background()
	extID := "clerk_123"
	userID := "user_1"

	t.Run("cache hit", func(t *testing.T) {
		mockUserCache.EXPECT().
			GetByExternalID(ctx, extID).
			Return(&domain.User{ID: userID, ExternalID: &extID}, nil)

		id, err := svc.GetByExternalID(ctx, extID)
		require.NoError(t, err)
		assert.Equal(t, userID, id)
	})

	t.Run("cache miss", func(t *testing.T) {
		mockUserCache.EXPECT().
			GetByExternalID(ctx, extID).
			Return(nil, errors.New("miss"))

		mockUserRepo.EXPECT().
			GetByExternalID(ctx, extID).
			Return(&domain.User{ID: userID, ExternalID: &extID}, nil)

		mockUserCache.EXPECT().
			SetByExternalID(ctx, gomock.Any()).
			Return(nil)

		id, err := svc.GetByExternalID(ctx, extID)
		require.NoError(t, err)
		assert.Equal(t, userID, id)
	})
}
