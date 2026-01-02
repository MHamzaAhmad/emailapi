package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v1 "github.com/emailapi/api/gen/v1"
	s3Mocks "github.com/emailapi/api/internal/external/s3/mocks"
	"github.com/emailapi/api/internal/repository/tinybird"
	tinybirdMocks "github.com/emailapi/api/internal/repository/tinybird/mocks"
	serviceMocks "github.com/emailapi/api/internal/service/mocks"
	webhookMocks "github.com/emailapi/api/internal/webhook/mocks"
)

func TestInboundEmailService_HandleNotification(t *testing.T) {
	t.Run("success inbound reply", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
		mockS3Client := s3Mocks.NewMockClient(ctrl)
		mockAnalytics := serviceMocks.NewMockAnalytics(ctrl)
		mockWebhook := webhookMocks.NewMockSender(ctrl)
		mockEmailRepo := tinybirdMocks.NewMockEmailRepositoryInterface(ctrl)

		mockAnalytics.EXPECT().Email().Return(mockEmailRepo).AnyTimes()
		mockS3Factory.EXPECT().Bucket(gomock.Any()).Return(mockS3Client).AnyTimes()

		svc := NewInboundEmailService(mockS3Factory, mockAnalytics, mockWebhook)
		ctx := context.Background()

		message := `{
			"notificationType": "Received",
			"receipt": {
				"action": {
					"type": "S3",
					"bucketName": "inbound-bucket",
					"objectKey": "emails/12345"
				}
			},
			"mail": {
				"messageId": "msg_inbound_1",
				"source": "sender@external.com",
				"destination": ["reply@myplatform.com"]
			}
		}`

		// Raw email content (replying to previous email)
		rawEmail := []byte("From: sender@external.com\r\n" +
			"To: reply@myplatform.com\r\n" +
			"Subject: Re: Hello\r\n" +
			"Message-ID: <msg_inbound_1>\r\n" +
			"In-Reply-To: <original_msg_1>\r\n" +
			"References: <original_msg_1>\r\n" +
			"\r\n" +
			"This is a reply body.")

		// Expectations
		mockS3Client.EXPECT().
			Download(ctx, "emails/12345").
			Return(rawEmail, nil)

			// Routing lookup for original email
		mockEmailRepo.EXPECT().
			LookupRouting(ctx, "original_msg_1").
			Return(&tinybird.EmailRouting{UserID: "user_1", EmailID: "email_sent_1", MessageID: "original_msg_1"}, nil)

		// Insert new routing
		mockEmailRepo.EXPECT().
			InsertRouting(ctx, "msg_inbound_1", gomock.Any(), "user_1").
			Return(nil)

		// Log activity (LogEmailEvent)
		mockEmailRepo.EXPECT().
			LogEmailEvent(ctx, "user_1", "email_sent_1", "replied", "success", gomock.Any(), gomock.Any()).
			Return(nil)

		// Webhook delivery
		mockWebhook.EXPECT().
			SendEmailReplied(ctx, "user_1", gomock.Any()).
			DoAndReturn(func(_ context.Context, uid string, evt *v1.EmailRepliedEvent) {
				assert.Equal(t, "msg_inbound_1", evt.MessageId)
				assert.Equal(t, "sender@external.com", evt.From)
				assert.Equal(t, "This is a reply body.", evt.Body)
				assert.Equal(t, "original_msg_1", evt.InReplyTo)
				assert.Equal(t, "email_sent_1", evt.ParentEmailId)
			})

		err := svc.HandleSNSNotification(ctx, &SNSNotificationInput{
			Type:    "Notification",
			Message: message,
		})
		require.NoError(t, err)
	})

	t.Run("auto response ignored", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
		mockS3Client := s3Mocks.NewMockClient(ctrl)
		mockAnalytics := serviceMocks.NewMockAnalytics(ctrl)
		mockWebhook := webhookMocks.NewMockSender(ctrl)
		mockEmailRepo := tinybirdMocks.NewMockEmailRepositoryInterface(ctrl)

		mockAnalytics.EXPECT().Email().Return(mockEmailRepo).AnyTimes()
		mockS3Factory.EXPECT().Bucket(gomock.Any()).Return(mockS3Client).AnyTimes()

		svc := NewInboundEmailService(mockS3Factory, mockAnalytics, mockWebhook)
		ctx := context.Background()

		message := `{ "notificationType": "Received", "receipt": { "action": { "type": "S3", "objectKey": "emails/ooo" } } }`

		// OOO email
		rawEmail := []byte("From: sender@external.com\r\n" +
			"To: reply@myplatform.com\r\n" +
			"Subject: Out of Office\r\n" +
			"Auto-Submitted: auto-replied\r\n" + // Triggers auto-response detector
			"In-Reply-To: <original_msg_1>\r\n" +
			"\r\n" +
			"I am on vacation.")

		mockS3Client.EXPECT().
			Download(ctx, "emails/ooo").
			Return(rawEmail, nil)

		mockEmailRepo.EXPECT().
			LookupRouting(ctx, "original_msg_1").
			Return(&tinybird.EmailRouting{UserID: "user_1", EmailID: "email_sent_1", MessageID: "original_msg_1"}, nil)

		mockEmailRepo.EXPECT().
			InsertRouting(ctx, gomock.Any(), gomock.Any(), "user_1").
			Return(nil)

		// Log as auto_response / ignored
		mockEmailRepo.EXPECT().
			LogEmailEvent(ctx, "user_1", "email_sent_1", "auto_response", "ignored", gomock.Any(), gomock.Any()).
			Return(nil)

		// Should NOT call webhook
		// Implicit: no expectation set for mockWebhook

		err := svc.HandleSNSNotification(ctx, &SNSNotificationInput{
			Type:    "Notification",
			Message: message,
		})
		require.NoError(t, err)
	})
}
