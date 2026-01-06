package handlers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/events"
	"github.com/emailapi/api/internal/events/handlers"
	"github.com/emailapi/api/internal/events/handlers/mocks"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/repository/tinybird"
	webhookMocks "github.com/emailapi/api/internal/webhook/mocks"
)

func TestBounceHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalytics := mocks.NewMockAnalytics(ctrl)
	mockActivityLogger := mocks.NewMockActivityLogger(ctrl)
	mockEmailRouter := mocks.NewMockEmailRouter(ctrl)
	mockSuppression := mocks.NewMockSuppressionManager(ctrl)
	mockReputation := mocks.NewMockReputationRecorder(ctrl)
	mockWebhook := webhookMocks.NewMockSender(ctrl)

	deps := &handlers.Dependencies{
		Analytics:     mockAnalytics,
		WebhookSender: mockWebhook,
		SuppressRepo:  mockSuppression,
		ReputationSvc: mockReputation,
	}

	handler := handlers.NewBounceHandler(deps)
	ctx := context.Background()

	t.Run("missing bounce data", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Bounce",
			Mail:      events.MailInfo{MessageId: "msg_1"},
			Bounce:    nil,
		}

		err := handler.Handle(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing bounce data")
	})

	t.Run("hard bounce success", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Bounce",
			Mail:      events.MailInfo{MessageId: "msg_1"},
			Bounce: &events.Bounce{
				BounceType:    "Permanent",
				BounceSubType: "General",
				BouncedRecipients: []events.BouncedRecipient{
					{EmailAddress: "test@example.com", DiagnosticCode: "550 User not found"},
				},
			},
		}

		// Hard bounce adds to global suppression
		mockSuppression.EXPECT().
			Add(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, entry *suppression.Entry) error {
				assert.Equal(t, "", entry.UserID) // Global
				assert.Equal(t, suppression.ReasonBounceHard, entry.Reason)
				return nil
			})

		// Lookup routing
		mockAnalytics.EXPECT().Email().Return(mockEmailRouter)
		mockEmailRouter.EXPECT().
			LookupRouting(ctx, "msg_1").
			Return(&tinybird.EmailRouting{UserID: "user_1", EmailID: "email_1"}, nil)

		// Log activity
		mockAnalytics.EXPECT().Activity().Return(mockActivityLogger)
		mockActivityLogger.EXPECT().
			Log(ctx, "user_1", "email", "email_1", "bounced", "failed", gomock.Any(), gomock.Any()).
			Return(nil)

		// Send webhook
		mockWebhook.EXPECT().
			SendEmailBounced(ctx, "user_1", gomock.Any())

		// Reputation (async, so we don't verify here)
		mockReputation.EXPECT().
			RecordBounceIncident(gomock.Any(), "user_1", "msg_1", "Permanent", "General", gomock.Any()).
			Return(nil).AnyTimes()

		err := handler.Handle(ctx, event)
		require.NoError(t, err)
	})

	t.Run("soft bounce success", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Bounce",
			Mail:      events.MailInfo{MessageId: "msg_2"},
			Bounce: &events.Bounce{
				BounceType:    "Transient",
				BounceSubType: "MailboxFull",
				BouncedRecipients: []events.BouncedRecipient{
					{EmailAddress: "full@example.com"},
				},
			},
		}

		// Lookup routing first (no global suppression for soft bounce)
		mockAnalytics.EXPECT().Email().Return(mockEmailRouter)
		mockEmailRouter.EXPECT().
			LookupRouting(ctx, "msg_2").
			Return(&tinybird.EmailRouting{UserID: "user_1", EmailID: "email_2"}, nil)

		// Soft bounce adds to user suppression
		mockSuppression.EXPECT().
			Add(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, entry *suppression.Entry) error {
				assert.Equal(t, "user_1", entry.UserID) // Per-user
				assert.Equal(t, suppression.ReasonBounceSoft, entry.Reason)
				return nil
			})

		// Log activity
		mockAnalytics.EXPECT().Activity().Return(mockActivityLogger)
		mockActivityLogger.EXPECT().
			Log(ctx, "user_1", "email", "email_2", "bounced", "failed", gomock.Any(), gomock.Any()).
			Return(nil)

		// Send webhook
		mockWebhook.EXPECT().
			SendEmailBounced(ctx, "user_1", gomock.Any())

		// Reputation (async)
		mockReputation.EXPECT().
			RecordBounceIncident(gomock.Any(), "user_1", "msg_2", "Transient", "MailboxFull", gomock.Any()).
			Return(nil).AnyTimes()

		err := handler.Handle(ctx, event)
		require.NoError(t, err)
	})

	t.Run("no routing - suppression only", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Bounce",
			Mail:      events.MailInfo{MessageId: "msg_3"},
			Bounce: &events.Bounce{
				BounceType:    "Permanent",
				BounceSubType: "NoEmail",
				BouncedRecipients: []events.BouncedRecipient{
					{EmailAddress: "dead@example.com"},
				},
			},
		}

		// Hard bounce adds to global suppression
		mockSuppression.EXPECT().
			Add(ctx, gomock.Any()).
			Return(nil)

		// Lookup routing fails
		mockAnalytics.EXPECT().Email().Return(mockEmailRouter)
		mockEmailRouter.EXPECT().
			LookupRouting(ctx, "msg_3").
			Return(nil, errors.New("not found"))

		// No activity log, webhook, or reputation since routing failed
		err := handler.Handle(ctx, event)
		require.NoError(t, err)
	})
}

func TestComplaintHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalytics := mocks.NewMockAnalytics(ctrl)
	mockActivityLogger := mocks.NewMockActivityLogger(ctrl)
	mockEmailRouter := mocks.NewMockEmailRouter(ctrl)
	mockSuppression := mocks.NewMockSuppressionManager(ctrl)
	mockReputation := mocks.NewMockReputationRecorder(ctrl)
	mockWebhook := webhookMocks.NewMockSender(ctrl)

	deps := &handlers.Dependencies{
		Analytics:     mockAnalytics,
		WebhookSender: mockWebhook,
		SuppressRepo:  mockSuppression,
		ReputationSvc: mockReputation,
	}

	handler := handlers.NewComplaintHandler(deps)
	ctx := context.Background()

	t.Run("missing complaint data", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Complaint",
			Mail:      events.MailInfo{MessageId: "msg_1"},
			Complaint: nil,
		}

		err := handler.Handle(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing complaint data")
	})

	t.Run("success", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Complaint",
			Mail:      events.MailInfo{MessageId: "msg_1"},
			Complaint: &events.Complaint{
				ComplaintFeedbackType: "abuse",
				ComplainedRecipients: []events.ComplainedRecipient{
					{EmailAddress: "complainer@example.com"},
				},
			},
		}

		// Add to global suppression
		mockSuppression.EXPECT().
			Add(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, entry *suppression.Entry) error {
				assert.Equal(t, "", entry.UserID) // Global
				assert.Equal(t, suppression.ReasonComplaint, entry.Reason)
				return nil
			})

		// Lookup routing
		mockAnalytics.EXPECT().Email().Return(mockEmailRouter)
		mockEmailRouter.EXPECT().
			LookupRouting(ctx, "msg_1").
			Return(&tinybird.EmailRouting{UserID: "user_1", EmailID: "email_1"}, nil)

		// Log activity
		mockAnalytics.EXPECT().Activity().Return(mockActivityLogger)
		mockActivityLogger.EXPECT().
			Log(ctx, "user_1", "email", "email_1", "complained", "failed", gomock.Any(), gomock.Any()).
			Return(nil)

		// Send webhook
		mockWebhook.EXPECT().
			SendEmailComplained(ctx, "user_1", gomock.Any())

		// Reputation (async)
		mockReputation.EXPECT().
			RecordComplaintIncident(gomock.Any(), "user_1", "msg_1", "abuse", gomock.Any()).
			Return(nil).AnyTimes()

		err := handler.Handle(ctx, event)
		require.NoError(t, err)
	})
}

func TestDeliveryHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalytics := mocks.NewMockAnalytics(ctrl)
	mockActivityLogger := mocks.NewMockActivityLogger(ctrl)
	mockEmailRouter := mocks.NewMockEmailRouter(ctrl)
	mockWebhook := webhookMocks.NewMockSender(ctrl)

	deps := &handlers.Dependencies{
		Analytics:     mockAnalytics,
		WebhookSender: mockWebhook,
	}

	handler := handlers.NewDeliveryHandler(deps)
	ctx := context.Background()

	t.Run("missing delivery data", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Delivery",
			Mail:      events.MailInfo{MessageId: "msg_1"},
			Delivery:  nil,
		}

		err := handler.Handle(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing delivery data")
	})

	t.Run("success", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Delivery",
			Mail:      events.MailInfo{MessageId: "msg_1"},
			Delivery: &events.Delivery{
				Recipients:   []string{"recipient@example.com"},
				Timestamp:    "2024-01-01T00:00:00Z",
				SmtpResponse: "250 OK",
			},
		}

		// Lookup routing
		mockAnalytics.EXPECT().Email().Return(mockEmailRouter)
		mockEmailRouter.EXPECT().
			LookupRouting(ctx, "msg_1").
			Return(&tinybird.EmailRouting{UserID: "user_1", EmailID: "email_1"}, nil)

		// Log activity
		mockAnalytics.EXPECT().Activity().Return(mockActivityLogger)
		mockActivityLogger.EXPECT().
			Log(ctx, "user_1", "email", "email_1", "delivered", "success", gomock.Any(), gomock.Any()).
			Return(nil)

		// Send webhook
		mockWebhook.EXPECT().
			SendEmailDelivered(ctx, "user_1", gomock.Any())

		err := handler.Handle(ctx, event)
		require.NoError(t, err)
	})

	t.Run("no routing - silent skip", func(t *testing.T) {
		event := &events.SESEvent{
			EventType: "Delivery",
			Mail:      events.MailInfo{MessageId: "msg_2"},
			Delivery: &events.Delivery{
				Recipients: []string{"recipient@example.com"},
			},
		}

		// Lookup routing fails
		mockAnalytics.EXPECT().Email().Return(mockEmailRouter)
		mockEmailRouter.EXPECT().
			LookupRouting(ctx, "msg_2").
			Return(nil, errors.New("not found"))

		// No further calls expected
		err := handler.Handle(ctx, event)
		require.NoError(t, err)
	})
}
