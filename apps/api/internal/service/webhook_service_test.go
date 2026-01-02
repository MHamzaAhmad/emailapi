package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	svixMocks "github.com/emailapi/api/internal/external/svix/mocks"
)

func TestWebhookService_GetAppPortalAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvix := svixMocks.NewMockClient(ctrl)
	svc := NewWebhookService(mockSvix)
	ctx := context.Background()
	userID := "user_123"
	userName := "User user_123"

	t.Run("success", func(t *testing.T) {
		expectedURL := "https://app.svix.com/portal/..."
		expectedToken := "test-token"

		mockSvix.EXPECT().
			EnsureApp(ctx, userID, userName).
			Return(nil)

		mockSvix.EXPECT().
			GetAppPortalAccess(ctx, userID).
			Return(expectedURL, expectedToken, nil)

		url, token, err := svc.GetAppPortalAccess(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, expectedURL, url)
		assert.Equal(t, expectedToken, token)
	})

	t.Run("ensure app error", func(t *testing.T) {
		mockSvix.EXPECT().
			EnsureApp(ctx, userID, userName).
			Return(errors.New("svix error"))

		url, token, err := svc.GetAppPortalAccess(ctx, userID)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "failed to ensure svix app")
	})

	t.Run("get portal access error", func(t *testing.T) {
		mockSvix.EXPECT().
			EnsureApp(ctx, userID, userName).
			Return(nil)

		mockSvix.EXPECT().
			GetAppPortalAccess(ctx, userID).
			Return("", "", errors.New("access error"))

		url, token, err := svc.GetAppPortalAccess(ctx, userID)
		assert.Error(t, err)
		assert.Empty(t, url)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "failed to get app portal access")
	})
}
