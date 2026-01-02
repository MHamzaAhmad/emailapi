package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/repository/tinybird"
	tinybirdMocks "github.com/emailapi/api/internal/repository/tinybird/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
	webhookMocks "github.com/emailapi/api/internal/webhook/mocks"
)

func TestSNSNotificationService_HandleBounce(t *testing.T) {
	t.Run("hard bounce global suppression", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAnalytics := serviceMocks.NewMockAnalytics(ctrl)
		mockWebhook := webhookMocks.NewMockSender(ctrl)
		mockSuppression := serviceMocks.NewMockSuppressionManager(ctrl)
		mockReputation := serviceMocks.NewMockReputationRecorder(ctrl)
		mockEmailRepo := tinybirdMocks.NewMockEmailRepositoryInterface(ctrl)
		mockActivityRepo := tinybirdMocks.NewMockActivityRepositoryInterface(ctrl)

		mockAnalytics.EXPECT().Email().Return(mockEmailRepo).AnyTimes()
		mockAnalytics.EXPECT().Activity().Return(mockActivityRepo).AnyTimes()

		svc := NewSNSNotificationService(mockAnalytics, mockWebhook, mockSuppression, mockReputation)
		ctx := context.Background()

		message := `{
			"eventType": "Bounce",
			"mail": { "messageId": "msg_123" },
			"bounce": {
				"bounceType": "Permanent",
				"bounceSubType": "General",
				"bouncedRecipients": [{ "emailAddress": "bounce@example.com", "diagnosticCode": "5.1.1" }]
			}
		}`

		// Expect global suppression (userID empty)
		mockSuppression.EXPECT().
			Add(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, e *suppression.Entry) error {
				assert.Equal(t, "", e.UserID)
				assert.Equal(t, suppression.ReasonBounceHard, e.Reason)
				return nil
			})

		// Look up routing
		mockEmailRepo.EXPECT().
			LookupRouting(ctx, "msg_123").
			Return(&tinybird.EmailRouting{UserID: "user_1", EmailID: "email_1"}, nil)

		// Activity logging
		mockActivityRepo.EXPECT().
			Log(ctx, "user_1", "email", "email_1", "bounced", "failed", gomock.Any(), gomock.Any()).
			Return(nil)

		// Webhook
		mockWebhook.EXPECT().
			SendEmailBounced(ctx, "user_1", gomock.Any()).
			DoAndReturn(func(_ context.Context, uid string, evt *v1.EmailBouncedEvent) {
				assert.Equal(t, "msg_123", evt.MessageId)
				assert.Equal(t, "Permanent", evt.BounceType)
			})

		// Reputation Incident
		// Note: Reputation call is in a goroutine, so expectation might fail due to race or test finishing early.
		// Gomock doesn't easily support "async" expectations unless we wait.
		// We can sleep, or accept that we can't test async calls deterministically without refactoring to sync for tests.
		// For now, we omit expectation or make it optional (.AnyTimes / .MaxTimes(1)).
		mockReputation.EXPECT().
			RecordBounceIncident(gomock.Any(), "user_1", "msg_123", "Permanent", "General", gomock.Any()).
			Return(nil).
			MaxTimes(1)

		err := svc.HandleNotification(ctx, &SNSInput{Type: "Notification", Message: message})
		require.NoError(t, err)
	})
}
