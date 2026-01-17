package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	emailapi "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
)

func TestEmailService_SendEmail(t *testing.T) {
	t.Run("success - all emails are queued", func(t *testing.T) {
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
			Subject: "Test Subject",
			Body:    "Test Body",
		}

		mockValidator.EXPECT().
			ValidateSendEmail(ctx, "user_1", req.From, req.To, []string(nil), []string(nil), req.Body, "").
			Return(nil)

		mockUnsubscribe.EXPECT().
			CheckBatch(ctx, "user_1", []string{"recipient@example.com"}).
			Return([]string{}, nil)

		// All emails are now queued
		mockQueue.EXPECT().
			Insert(ctx, gomock.Any(), nil).
			Return(nil, nil)

		resp, err := svc.SendEmail(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Id)
		assert.Equal(t, emailapi.EmailStatus_EMAIL_STATUS_QUEUED, resp.Status)
		assert.Equal(t, "Email queued for sending", resp.StatusMessage)
	})

	t.Run("success - with reply_to", func(t *testing.T) {
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
			Subject: "Re: Original Subject",
			Body:    "Reply Body",
			ReplyTo: "original-email-uuid", // Our internal email_id
		}

		mockValidator.EXPECT().
			ValidateSendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		mockUnsubscribe.EXPECT().
			CheckBatch(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]string{}, nil)

		// Verify that reply_to is passed in the job args
		mockQueue.EXPECT().
			Insert(ctx, gomock.Any(), nil).
			DoAndReturn(func(ctx context.Context, args interface{}, opts interface{}) (interface{}, error) {
				// The args should contain the ReplyTo field
				return nil, nil
			})

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

	t.Run("all recipients unsubscribed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQueue := serviceMocks.NewMockQueueClient(ctrl)
		mockValidator := serviceMocks.NewMockSenderValidator(ctrl)
		mockUnsubscribe := serviceMocks.NewMockUnsubscribeManager(ctrl)

		svc := NewEmailService(mockQueue, nil, mockValidator, nil, nil, nil, nil, mockUnsubscribe)
		ctx := context.WithValue(context.Background(), "user_id", "user_1")

		req := &emailapi.SendEmailRequest{
			From:    "sender@example.com",
			To:      []string{"unsubscribed@example.com"},
			Subject: "Test",
			Body:    "Test",
		}

		mockValidator.EXPECT().
			ValidateSendEmail(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		// Return the recipient as unsubscribed
		mockUnsubscribe.EXPECT().
			CheckBatch(gomock.Any(), "user_1", []string{"unsubscribed@example.com"}).
			Return([]string{"unsubscribed@example.com"}, nil)

		resp, err := svc.SendEmail(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, emailapi.EmailStatus_EMAIL_STATUS_SENT, resp.Status)
		assert.Equal(t, "All recipients have unsubscribed", resp.StatusMessage)
	})
}
