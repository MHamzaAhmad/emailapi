package worker

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	s3Mocks "github.com/emailapi/api/internal/external/s3/mocks"
	sesMocks "github.com/emailapi/api/internal/external/ses/mocks"
	tinybirdMocks "github.com/emailapi/api/internal/repository/tinybird/mocks"
)

func TestEmailWorker_Work_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSES := sesMocks.NewMockClient(ctrl)
	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockTBRepo := tinybirdMocks.NewMockEmailRepositoryInterface(ctrl)

	worker := NewEmailWorker(mockSES, mockS3Factory, mockTBRepo, nil, nil, nil)

	ctx := context.Background()
	job := &river.Job[SendEmailArgs]{
		Args: SendEmailArgs{
			EmailID: "email_1",
			UserID:  "user_1",
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Test Subject",
			Body:    "Test Body",
			HTML:    "<p>Test Body</p>",
		},
	}

	// Expect SES SendEmail
	mockSES.EXPECT().SendEmail(ctx, gomock.Any()).Return(&sesv2.SendEmailOutput{MessageId: Point("msg_ses_123")}, nil).Times(1)

	// Expect Tinybird routing insert
	mockTBRepo.EXPECT().InsertRouting(ctx, "msg_ses_123", "email_1", "user_1").Return(nil)

	// Expect Activity Log (Sent)
	mockTBRepo.EXPECT().LogActivity(ctx, "user_1", "email", "email_1", "sent", "success", gomock.Any(), gomock.Any()).Return(nil)

	err := worker.Work(ctx, job)
	require.NoError(t, err)
}

func TestEmailWorker_Work_DryRun(t *testing.T) {
	worker := NewEmailWorker(nil, nil, nil, nil, nil, nil)
	ctx := context.Background()
	job := &river.Job[SendEmailArgs]{
		Args: SendEmailArgs{
			EmailID: "email_dry",
			DryRun:  true,
		},
	}
	err := worker.Work(ctx, job)
	require.NoError(t, err)
}

func TestEmailWorker_Work_Split(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSES := sesMocks.NewMockClient(ctrl)
	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockTBRepo := tinybirdMocks.NewMockEmailRepositoryInterface(ctrl)

	worker := NewEmailWorker(mockSES, mockS3Factory, mockTBRepo, nil, nil, nil)

	ctx := context.Background()

	// Prepare args for splitting
	args := SendEmailArgs{
		EmailID:                "email_split",
		UserID:                 "user_1",
		From:                   "sender@example.com",
		To:                     []string{"r1@example.com", "r2@example.com"},
		Subject:                "Split Subject",
		Body:                   "Body with {{unsubscribe_link}}",
		HTML:                   "<p>Body with {{unsubscribe_link}}</p>",
		UnsubscribeTokenSecret: "secret",
		UnsubscribeBaseURL:     "https://api.example.com",
	}

	job := &river.Job[SendEmailArgs]{Args: args}

	// S3 Factory
	// Attachments are nil, but code might call Bucket/Download if keys are present?
	// Args has no attachment keys. So no calls.

	// Expect 2 interactions for 2 recipients

	// 1. Recipient 1
	mockSES.EXPECT().SendEmail(ctx, gomock.Any()).Return(&sesv2.SendEmailOutput{MessageId: Point("msg_1")}, nil).Times(1)
	mockTBRepo.EXPECT().InsertRouting(ctx, "msg_1", "email_split", "user_1").Return(nil)
	mockTBRepo.EXPECT().LogActivity(ctx, "user_1", "email", "email_split", "sent", "success", gomock.Any(), gomock.Any()).Return(nil)

	// 2. Recipient 2
	mockSES.EXPECT().SendEmail(ctx, gomock.Any()).Return(&sesv2.SendEmailOutput{MessageId: Point("msg_2")}, nil).Times(1)
	mockTBRepo.EXPECT().InsertRouting(ctx, "msg_2", "email_split", "user_1").Return(nil)
	mockTBRepo.EXPECT().LogActivity(ctx, "user_1", "email", "email_split", "sent", "success", gomock.Any(), gomock.Any()).Return(nil)

	err := worker.Work(ctx, job)
	require.NoError(t, err)
}

func Point[T any](v T) *T { return &v }
