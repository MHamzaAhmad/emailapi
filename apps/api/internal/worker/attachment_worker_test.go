package worker

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/external/s3"
	s3Mocks "github.com/emailapi/api/internal/external/s3/mocks"
	"github.com/emailapi/api/internal/worker/mocks"
)

func TestAttachmentWorker_Work_DryRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockRiver := mocks.NewMockRiverClient(ctrl)

	worker := NewAttachmentWorker(mockS3Factory, mockRiver)
	ctx := context.Background()

	job := &river.Job[ProcessAttachmentsArgs]{
		Args: ProcessAttachmentsArgs{
			EmailID: "email_dry",
			DryRun:  true,
			Attachments: []AttachmentSource{
				{Filename: "test.txt", ContentType: "text/plain", Base64Content: "SGVsbG8="},
			},
		},
	}

	// Expect River Insert with DryRun=true and fake keys
	mockRiver.EXPECT().Insert(ctx, gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
		sendArgs, ok := args.(SendEmailArgs)
		require.True(t, ok)
		assert.True(t, sendArgs.DryRun)
		assert.Equal(t, "email_dry", sendArgs.EmailID)
		assert.Len(t, sendArgs.AttachmentKeys, 1)
		assert.Contains(t, sendArgs.AttachmentKeys[0].S3Key, "dry-run")
		return &rivertype.JobInsertResult{}, nil
	})

	err := worker.Work(ctx, job)
	require.NoError(t, err)
}

func TestAttachmentWorker_Work_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockS3Client := s3Mocks.NewMockClient(ctrl)
	mockRiver := mocks.NewMockRiverClient(ctrl)

	worker := NewAttachmentWorker(mockS3Factory, mockRiver)
	ctx := context.Background()

	job := &river.Job[ProcessAttachmentsArgs]{
		Args: ProcessAttachmentsArgs{
			EmailID: "email_real",
			UserID:  "user_1",
			Attachments: []AttachmentSource{
				{Filename: "test.txt", ContentType: "text/plain", Base64Content: "SGVsbG8="},
			},
		},
	}

	// Expect S3 bucket retrieval and upload
	mockS3Factory.EXPECT().Bucket(s3.BucketAttachments).Return(mockS3Client).AnyTimes()
	// Wait, the code calls `w.s3Factory.Bucket(s3.BucketAttachments)` inside the loop?
	// Yes: `w.s3Factory.Bucket(s3.BucketAttachments).UploadAttachment` inside goroutine.
	// Since there is 1 attachment, it should be called once.
	// However, it runs in goroutine.

	mockS3Client.EXPECT().UploadAttachment(ctx, gomock.Any(), []byte("Hello"), "text/plain").Return(nil)

	// Expect River Insert with real Key
	mockRiver.EXPECT().Insert(ctx, gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
		sendArgs, ok := args.(SendEmailArgs)
		require.True(t, ok)
		assert.False(t, sendArgs.DryRun)
		assert.Equal(t, "email_real", sendArgs.EmailID)
		assert.Len(t, sendArgs.AttachmentKeys, 1)
		assert.Contains(t, sendArgs.AttachmentKeys[0].S3Key, "attachments/email_real/")
		return &rivertype.JobInsertResult{}, nil
	})

	err := worker.Work(ctx, job)
	require.NoError(t, err)
}
