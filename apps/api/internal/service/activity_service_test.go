package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
	"github.com/emailapi/api/internal/repository/tinybird/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
)

func TestActivityService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalytics := serviceMocks.NewMockAnalytics(ctrl)
	mockActivityRepo := mocks.NewMockActivityRepositoryInterface(ctrl)

	// Setup aggregator to return repo mock
	// We only expect this call if analytics is not nil
	mockAnalytics.EXPECT().Activity().Return(mockActivityRepo).AnyTimes()

	svc := NewActivityService(mockAnalytics)

	ctx := context.Background()
	userID := "user_123"

	t.Run("success", func(t *testing.T) {
		expectedLogs := []tbrepo.ActivityLog{
			{UserID: userID, Action: "logged_in", Timestamp: time.Now().Unix()},
		}
		expectedCount := 1

		mockActivityRepo.EXPECT().
			List(ctx, userID, tbrepo.ActivityFilters{}, 20, 0).
			Return(expectedLogs, expectedCount, nil)

		logs, count, err := svc.List(ctx, userID, ListFilters{}, 0, 0)
		require.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		assert.Equal(t, expectedLogs, logs)
	})

	t.Run("with filters", func(t *testing.T) {
		entityType := "user"
		action := "login"
		startTime := int64(100)
		endTime := int64(200)

		filters := ListFilters{
			EntityType: &entityType,
			Action:     &action,
			StartTime:  &startTime,
			EndTime:    &endTime,
		}

		repoFilters := tbrepo.ActivityFilters{
			EntityType: entityType,
			Action:     action,
			StartTime:  &startTime,
			EndTime:    &endTime,
		}

		mockActivityRepo.EXPECT().
			List(ctx, userID, repoFilters, 50, 10).
			Return([]tbrepo.ActivityLog{}, 0, nil)

		logs, count, err := svc.List(ctx, userID, filters, 50, 10)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Empty(t, logs)
	})

	t.Run("repo error", func(t *testing.T) {
		expectedErr := errors.New("db error")

		mockActivityRepo.EXPECT().
			List(ctx, userID, gomock.Any(), 20, 0).
			Return(nil, 0, expectedErr)

		logs, count, err := svc.List(ctx, userID, ListFilters{}, 0, 0)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 0, count)
		assert.Nil(t, logs)
	})

	t.Run("nil analytics", func(t *testing.T) {
		nilSvc := NewActivityService(nil)
		logs, count, err := nilSvc.List(ctx, userID, ListFilters{}, 0, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Nil(t, logs)
	})
}
