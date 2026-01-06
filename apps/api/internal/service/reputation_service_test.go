package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/domain"
	postgresMocks "github.com/emailapi/api/internal/repository/postgres/mocks"
	rediscache "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/repository/redis/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
)

func TestReputationService_CheckSendPermission(t *testing.T) {
	t.Run("allowed clean user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := serviceMocks.NewMockStore(ctrl)
		mockRepo := postgresMocks.NewMockReputationRepository(ctrl)
		mockCache := serviceMocks.NewMockCache(ctrl)
		mockReputationCache := mocks.NewMockReputationCacheInterface(ctrl)

		mockStore.EXPECT().Reputation().Return(mockRepo).AnyTimes()
		mockCache.EXPECT().Reputation().Return(mockReputationCache).AnyTimes()

		svc := NewReputationService(mockStore, nil, mockCache, nil)
		ctx := context.Background()

		// Cache miss
		mockReputationCache.EXPECT().Get(ctx, "user_1").Return(nil, assert.AnError)

		// DB check
		mockRepo.EXPECT().Get(ctx, "user_1").Return(&domain.UserReputation{
			IsSuspended: false,
			IsFlagged:   false,
		}, nil)

		// Cache set
		mockReputationCache.EXPECT().Set(ctx, "user_1", gomock.Any()).Return(nil)

		err := svc.CheckSendPermission(ctx, "user_1")
		require.NoError(t, err)
	})

	t.Run("suspended user cached", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := serviceMocks.NewMockCache(ctrl)
		mockReputationCache := mocks.NewMockReputationCacheInterface(ctrl)

		mockCache.EXPECT().Reputation().Return(mockReputationCache).AnyTimes()

		svc := NewReputationService(nil, nil, mockCache, nil)
		ctx := context.Background()

		// Cache hit - suspended
		mockReputationCache.EXPECT().Get(ctx, "user_suspended").Return(&rediscache.UserReputationStatus{
			IsSuspended: true,
		}, nil)

		err := svc.CheckSendPermission(ctx, "user_suspended")
		assert.Equal(t, ErrAccountSuspended, err)
	})
}

func TestReputationService_RecordBounceIncident(t *testing.T) {
	t.Run("record incident and queue eval", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := serviceMocks.NewMockStore(ctrl)
		mockRepo := postgresMocks.NewMockReputationRepository(ctrl)
		mockQueue := serviceMocks.NewMockQueueClient(ctrl)

		mockStore.EXPECT().Reputation().Return(mockRepo).AnyTimes()

		svc := NewReputationService(mockStore, mockQueue, nil, nil) // Skip analytics for simplicity or mock later if needed
		ctx := context.Background()

		recipients := []domain.BounceRecipient{
			{EmailAddress: "bounce@example.com", DiagnosticCode: "5.1.1 User unknown"},
		}

		// Insert Incident
		mockRepo.EXPECT().InsertIncident(ctx, gomock.Any()).Return(nil)

		// Queue Evaluation
		mockQueue.EXPECT().Insert(ctx, gomock.Any(), gomock.Any()).Return(nil, nil)

		err := svc.RecordBounceIncident(ctx, "user_1", "msg_1", "Permanent", "General", recipients)
		require.NoError(t, err)
	})
}
