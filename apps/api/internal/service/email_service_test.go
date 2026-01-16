package service

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	emailapi "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	sesMocks "github.com/emailapi/api/internal/external/ses/mocks"
	tinybirdMocks "github.com/emailapi/api/internal/repository/tinybird/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
	webhookMocks "github.com/emailapi/api/internal/webhook/mocks"
)

func TestEmailService_SendEmail(t *testing.T) {
	t.Run("success sync send", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQueue := serviceMocks.NewMockQueueClient(ctrl)
		mockAnalytics := serviceMocks.NewMockAnalytics(ctrl)
		mockValidator := serviceMocks.NewMockSenderValidator(ctrl)
		mockSES := sesMocks.NewMockClient(ctrl)
		mockWebhook := webhookMocks.NewMockSender(ctrl)
		mockUnsubscribe := serviceMocks.NewMockUnsubscribeManager(ctrl)
		mockTBEmail := tinybirdMocks.NewMockEmailRepositoryInterface(ctrl)

		mockAnalytics.EXPECT().Email().Return(mockTBEmail).AnyTimes()

		svc := NewEmailService(mockQueue, mockAnalytics, mockValidator, mockSES, mockWebhook, nil, nil, mockUnsubscribe)
		ctx := context.WithValue(context.Background(), "user_id", "user_1")

		req := &emailapi.SendEmailRequest{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Test Subject",
			Body:    "Test Body",
			Async:   false,
		}

		// Validation
		mockValidator.EXPECT().
			ValidateSendEmail(ctx, "user_1", req.From, req.To, []string(nil), []string(nil), req.Body, "").
			Return(nil)

		// Unsubscribe Check
		mockUnsubscribe.EXPECT().
			CheckBatch(ctx, "user_1", []string{"recipient@example.com"}).
			Return([]string{}, nil)

		// Send to SES
		mockSES.EXPECT().
			SendEmail(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, input *sesv2.SendEmailInput) (*sesv2.SendEmailOutput, error) {
				assert.Equal(t, "sender@example.com", *input.FromEmailAddress)
				assert.Equal(t, "recipient@example.com", input.Destination.ToAddresses[0])
				assert.Equal(t, "Test Subject", *input.Content.Simple.Subject.Data)
				return &sesv2.SendEmailOutput{
					MessageId: aws.String("ses_msg_id"),
				}, nil
			})

		// Log Activity
		mockTBEmail.EXPECT().
			LogEmailEvent(gomock.Any(), "user_1", gomock.Any(), "sent", "success", "Message ID: ses_msg_id", gomock.Any()).
			Return(nil).
			AnyTimes() // Async

		// Routing logic (Async)
		mockTBEmail.EXPECT().
			InsertRouting(gomock.Any(), "ses_msg_id", gomock.Any(), "user_1").
			Return(nil).
			AnyTimes()

		// Webhook
		mockWebhook.EXPECT().
			SendEmailSent(ctx, "user_1", gomock.Any()).
			Return().
			Times(1)

		resp, err := svc.SendEmail(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "ses_msg_id", resp.MessageId)
		assert.Equal(t, emailapi.EmailStatus_EMAIL_STATUS_SENT, resp.Status)

		// Wait for async goroutines
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("async queueing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQueue := serviceMocks.NewMockQueueClient(ctrl)
		mockAnalytics := serviceMocks.NewMockAnalytics(ctrl)
		mockValidator := serviceMocks.NewMockSenderValidator(ctrl)
		mockUnsubscribe := serviceMocks.NewMockUnsubscribeManager(ctrl)

		svc := NewEmailService(mockQueue, mockAnalytics, mockValidator, nil, nil, nil, nil, mockUnsubscribe)
		ctx := context.WithValue(context.Background(), "user_id", "user_1")

		req := &emailapi.SendEmailRequest{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Async Subject",
			Async:   true,
		}

		mockValidator.EXPECT().ValidateSendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		mockUnsubscribe.EXPECT().CheckBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return([]string{}, nil)

		// Expect Queue Insert
		mockQueue.EXPECT().
			Insert(ctx, gomock.Any(), nil).
			Return(nil, nil)

		resp, err := svc.SendEmail(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, emailapi.EmailStatus_EMAIL_STATUS_QUEUED, resp.Status)
	})

	t.Run("validation error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockValidator := serviceMocks.NewMockSenderValidator(ctrl)
		svc := NewEmailService(nil, nil, mockValidator, nil, nil, nil, nil, nil)
		ctx := context.WithValue(context.Background(), "user_id", "user_1")

		req := &emailapi.SendEmailRequest{}

		// Validator returns errors directly (e.g., ValidationErrors or AppError)
		// which are passed through without wrapping
		mockValidator.EXPECT().
			ValidateSendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(domain.ErrInvalidEmailSyntax.Clone().WithField("to[0]"))

		_, err := svc.SendEmail(ctx, req)
		assert.Error(t, err)
		appErr, ok := domain.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, domain.ErrInvalidEmailSyntax.Code, appErr.Code)
		assert.Equal(t, "to[0]", appErr.Field)
	})
}
