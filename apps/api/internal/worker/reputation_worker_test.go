package worker

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/domain"
	postgresMocks "github.com/emailapi/api/internal/repository/postgres/mocks"
	redisMocks "github.com/emailapi/api/internal/repository/redis/mocks"
	tinybirdMocks "github.com/emailapi/api/internal/repository/tinybird/mocks"
)

func TestReputationWorker_Work_Normal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReputationRepo := postgresMocks.NewMockReputationRepository(ctrl)
	mockActivityRepo := tinybirdMocks.NewMockActivityRepositoryInterface(ctrl)
	mockCache := redisMocks.NewMockReputationCacheInterface(ctrl)

	worker := NewReputationWorker(mockReputationRepo, mockActivityRepo, mockCache)
	ctx := context.Background()

	userID := "user_normal"
	job := &river.Job[EvaluateReputationArgs]{
		Args: EvaluateReputationArgs{UserID: userID},
	}

	// 1. Ensure exists
	mockReputationRepo.EXPECT().EnsureExists(ctx, userID).Return(nil)

	// 2. Count incidents (low counts)
	mockReputationRepo.EXPECT().CountIncidents(ctx, userID).Return(&domain.IncidentStats{
		HardBounces: 0, SoftBounces: 5, Complaints: 0,
		Bounces30d: 2, Complaints30d: 0,
	}, nil)

	// 3. Get current rep
	mockReputationRepo.EXPECT().Get(ctx, userID).Return(&domain.UserReputation{IsFlagged: false}, nil)

	// 4. Update stats (expect no flag)
	mockReputationRepo.EXPECT().UpdateStats(ctx, userID, gomock.Any()).DoAndReturn(func(_ context.Context, _ string, stats *domain.UserReputation) error {
		require.False(t, stats.IsFlagged)
		return nil
	})

	// 5. Invalidate cache
	mockCache.EXPECT().Delete(ctx, userID).Return(nil)

	err := worker.Work(ctx, job)
	require.NoError(t, err)
}

func TestReputationWorker_Work_Flagging(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReputationRepo := postgresMocks.NewMockReputationRepository(ctrl)
	mockActivityRepo := tinybirdMocks.NewMockActivityRepositoryInterface(ctrl)
	mockCache := redisMocks.NewMockReputationCacheInterface(ctrl)

	worker := NewReputationWorker(mockReputationRepo, mockActivityRepo, mockCache)
	ctx := context.Background()

	userID := "user_flagged"
	job := &river.Job[EvaluateReputationArgs]{
		Args: EvaluateReputationArgs{UserID: userID},
	}

	mockReputationRepo.EXPECT().EnsureExists(ctx, userID).Return(nil)

	// High complaints (flag threshold is 3 in 30d)
	mockReputationRepo.EXPECT().CountIncidents(ctx, userID).Return(&domain.IncidentStats{
		HardBounces: 0, SoftBounces: 0, Complaints: 4,
		Bounces30d: 0, Complaints30d: 4,
	}, nil)

	// Not previously flagged
	mockReputationRepo.EXPECT().Get(ctx, userID).Return(&domain.UserReputation{IsFlagged: false}, nil)

	// Expect update stats with IsFlagged=true
	mockReputationRepo.EXPECT().UpdateStats(ctx, userID, gomock.Any()).DoAndReturn(func(_ context.Context, _ string, stats *domain.UserReputation) error {
		require.True(t, stats.IsFlagged)
		require.Contains(t, stats.FlaggedReason, "High complaint rate")
		return nil
	})

	// Log activity (newly flagged)
	mockActivityRepo.EXPECT().Log(ctx, userID, "reputation", userID, "flagged", "warning", gomock.Any(), gomock.Any()).Return(nil)

	mockCache.EXPECT().Delete(ctx, userID).Return(nil)

	err := worker.Work(ctx, job)
	require.NoError(t, err)
}

func TestReputationWorker_Work_AutoSuspend(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReputationRepo := postgresMocks.NewMockReputationRepository(ctrl)
	mockActivityRepo := tinybirdMocks.NewMockActivityRepositoryInterface(ctrl)
	mockCache := redisMocks.NewMockReputationCacheInterface(ctrl)

	worker := NewReputationWorker(mockReputationRepo, mockActivityRepo, mockCache)
	ctx := context.Background()

	userID := "user_suspended"
	job := &river.Job[EvaluateReputationArgs]{
		Args: EvaluateReputationArgs{UserID: userID},
	}

	mockReputationRepo.EXPECT().EnsureExists(ctx, userID).Return(nil)

	// Critical hard bounces (suspend threshold is 20)
	mockReputationRepo.EXPECT().CountIncidents(ctx, userID).Return(&domain.IncidentStats{
		HardBounces: 25, SoftBounces: 0, Complaints: 0,
		Bounces30d: 25, Complaints30d: 0,
	}, nil)

	// Get current rep (not suspended yet)
	mockReputationRepo.EXPECT().Get(ctx, userID).Return(&domain.UserReputation{IsFlagged: false, IsSuspended: false}, nil)

	// Update stats first
	mockReputationRepo.EXPECT().UpdateStats(ctx, userID, gomock.Any()).Return(nil)

	// Expect Suspend call
	mockReputationRepo.EXPECT().Suspend(ctx, userID, "system", gomock.Any()).Return(nil)

	// Log critical activity
	mockActivityRepo.EXPECT().Log(ctx, userID, "reputation", userID, "auto_suspended", "critical", gomock.Any(), gomock.Any()).Return(nil)

	// Set cache suspended
	mockCache.EXPECT().SetSuspended(ctx, userID).Return(nil)

	err := worker.Work(ctx, job)
	require.NoError(t, err)
}
