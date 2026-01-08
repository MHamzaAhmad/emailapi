package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/domain"
	repoMocks "github.com/emailapi/api/internal/repository/postgres/mocks"
	redisMocks "github.com/emailapi/api/internal/repository/redis/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnsubscribeService_ProcessUnsubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repoMocks.NewMockUnsubscribeRepository(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)
	mockUnsubCache := redisMocks.NewMockUnsubscribeCacheInterface(ctrl)

	// Use real token service
	tokenSvc := NewUnsubscribeTokenService("secret")
	svc := NewUnsubscribeService(mockRepo, mockCache, tokenSvc, "http://localhost")

	ctx := context.Background()
	userID := "user_1"
	email := "test@example.com"
	emailID := "email_1"

	// Generate valid token data for test
	tokenData := &UnsubscribeTokenData{
		UserID:  userID,
		Email:   email,
		EmailID: emailID,
	}
	token, _ := tokenSvc.Encode(tokenData)

	t.Run("success", func(t *testing.T) {
		// Expect DB add
		mockRepo.EXPECT().
			Add(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, entry *domain.UnsubscribeEntry) error {
				assert.Equal(t, userID, entry.UserID)
				assert.Equal(t, HashEmail(email), entry.EmailHash)
				assert.Equal(t, emailID, entry.SourceEmailID)
				assert.Equal(t, domain.UnsubscribeSourceOneClick, entry.Source)
				return nil
			})

		// Expect Cache update
		mockCache.EXPECT().Unsubscribe().Return(mockUnsubCache)
		mockUnsubCache.EXPECT().
			Set(ctx, userID, HashEmail(email)).
			Return(nil)

		res, err := svc.ProcessUnsubscribe(ctx, token, domain.UnsubscribeSourceOneClick)
		require.NoError(t, err)
		assert.Equal(t, userID, res.UserID)
		assert.Equal(t, email, res.Email)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.ProcessUnsubscribe(ctx, "invalid-token", domain.UnsubscribeSourceLink)
		assert.Error(t, err)
		assert.Equal(t, domain.ErrInvalidToken, err)
	})

	t.Run("db error", func(t *testing.T) {
		mockRepo.EXPECT().
			Add(ctx, gomock.Any()).
			Return(errors.New("db fail"))

		_, err := svc.ProcessUnsubscribe(ctx, token, domain.UnsubscribeSourceLink)
		assert.Error(t, err)
	})
}

func TestUnsubscribeService_CheckBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repoMocks.NewMockUnsubscribeRepository(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)
	mockUnsubCache := redisMocks.NewMockUnsubscribeCacheInterface(ctrl)

	svc := NewUnsubscribeService(mockRepo, mockCache, nil, "")
	ctx := context.Background()
	userID := "user_1"
	emails := []string{"a@test.com", "b@test.com", "c@test.com"}

	// Hashes: a->hashA, b->hashB, c->hashC
	hashA := HashEmail("a@test.com")
	hashB := HashEmail("b@test.com")
	hashC := HashEmail("c@test.com")
	hashes := []string{hashA, hashB, hashC}

	t.Run("cache hit all", func(t *testing.T) {
		mockCache.EXPECT().Unsubscribe().Return(mockUnsubCache)
		mockUnsubCache.EXPECT().
			CheckBatch(ctx, userID, hashes).
			Return(hashes, nil)

		unsub, err := svc.CheckBatch(ctx, userID, emails)
		require.NoError(t, err)
		assert.Equal(t, 3, len(unsub))
		assert.ElementsMatch(t, emails, unsub)
	})

	t.Run("cache partial hit", func(t *testing.T) {
		// Cache returns A
		mockCache.EXPECT().Unsubscribe().Return(mockUnsubCache).Times(2) // Once for check, once for update
		mockUnsubCache.EXPECT().
			CheckBatch(ctx, userID, hashes).
			Return([]string{hashA}, nil)

		// DB checks remaining (B, C)
		mockRepo.EXPECT().
			CheckBatch(ctx, userID, []string{hashB, hashC}).
			Return([]string{hashC}, nil) // DB says C is unsubscribed too

		// Cache updated with C
		mockUnsubCache.EXPECT().
			SetBatch(ctx, userID, []string{hashC}).
			Return(nil)

		unsub, err := svc.CheckBatch(ctx, userID, emails)
		require.NoError(t, err)
		assert.Equal(t, 2, len(unsub))
		assert.ElementsMatch(t, []string{"a@test.com", "c@test.com"}, unsub)
	})

	t.Run("cache miss", func(t *testing.T) {
		mockCache.EXPECT().Unsubscribe().Return(mockUnsubCache)
		mockUnsubCache.EXPECT().
			CheckBatch(ctx, userID, hashes).
			Return(nil, nil)

		mockRepo.EXPECT().
			CheckBatch(ctx, userID, hashes).
			Return(nil, nil)

		unsub, err := svc.CheckBatch(ctx, userID, emails)
		require.NoError(t, err)
		assert.Empty(t, unsub)
	})
}

func TestUnsubscribeService_Resubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repoMocks.NewMockUnsubscribeRepository(ctrl)
	mockCache := serviceMocks.NewMockCache(ctrl)
	mockUnsubCache := redisMocks.NewMockUnsubscribeCacheInterface(ctrl)

	svc := NewUnsubscribeService(mockRepo, mockCache, nil, "")
	ctx := context.Background()
	userID := "user_1"
	email := "test@example.com"
	hash := HashEmail(email)

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().
			Delete(ctx, userID, hash).
			Return(nil)

		mockCache.EXPECT().Unsubscribe().Return(mockUnsubCache)
		mockUnsubCache.EXPECT().
			Delete(ctx, userID, hash).
			Return(nil)

		err := svc.Resubscribe(ctx, userID, email)
		assert.NoError(t, err)
	})
}
