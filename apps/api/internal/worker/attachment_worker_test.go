package worker

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/external/s3"
	s3Mocks "github.com/emailapi/api/internal/external/s3/mocks"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	redisMocks "github.com/emailapi/api/internal/repository/redis/mocks"
)

func TestAttachmentWorker_Work_DryRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockPendingCache := redisMocks.NewMockPendingAttachmentCacheInterface(ctrl)

	worker := NewAttachmentWorker(mockS3Factory, mockPendingCache)
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

	// Dry run should just log and return nil, no S3 or cache calls
	err := worker.Work(ctx, job)
	require.NoError(t, err)
}

func TestAttachmentWorker_Work_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockS3Client := s3Mocks.NewMockClient(ctrl)
	mockPendingCache := redisMocks.NewMockPendingAttachmentCacheInterface(ctrl)

	worker := NewAttachmentWorker(mockS3Factory, mockPendingCache)
	ctx := context.Background()

	job := &river.Job[ProcessAttachmentsArgs]{
		Args: ProcessAttachmentsArgs{
			EmailID: "email_real",
			UserID:  "user_1",
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Test",
			Attachments: []AttachmentSource{
				{Filename: "test.txt", ContentType: "text/plain", Base64Content: "SGVsbG8="},
			},
		},
	}

	// Expect S3 upload
	mockS3Factory.EXPECT().Bucket(s3.BucketAttachments).Return(mockS3Client).AnyTimes()
	mockS3Client.EXPECT().UploadAttachment(ctx, gomock.Any(), []byte("Hello"), "text/plain").Return(nil)

	// Expect pending cache store
	mockPendingCache.EXPECT().Store(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, data *redisrepo.PendingAttachmentData) error {
		assert.Equal(t, "email_real", data.EmailID)
		assert.Equal(t, "user_1", data.UserID)
		assert.Len(t, data.AttachmentKeys, 1)
		assert.Equal(t, 1, data.PendingCount)
		return nil
	})

	err := worker.Work(ctx, job)
	require.NoError(t, err)
}

func TestAttachmentWorker_Work_UploadError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockS3Client := s3Mocks.NewMockClient(ctrl)
	mockPendingCache := redisMocks.NewMockPendingAttachmentCacheInterface(ctrl)

	worker := NewAttachmentWorker(mockS3Factory, mockPendingCache)
	ctx := context.Background()

	job := &river.Job[ProcessAttachmentsArgs]{
		Args: ProcessAttachmentsArgs{
			EmailID: "email_fail",
			UserID:  "user_1",
			Attachments: []AttachmentSource{
				{Filename: "test.txt", ContentType: "text/plain", Base64Content: "SGVsbG8="},
			},
		},
	}

	// Expect S3 upload failure
	mockS3Factory.EXPECT().Bucket(s3.BucketAttachments).Return(mockS3Client).AnyTimes()
	mockS3Client.EXPECT().UploadAttachment(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(assert.AnError)

	err := worker.Work(ctx, job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upload")
}

func TestAttachmentWorker_Work_CacheError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockS3Client := s3Mocks.NewMockClient(ctrl)
	mockPendingCache := redisMocks.NewMockPendingAttachmentCacheInterface(ctrl)

	worker := NewAttachmentWorker(mockS3Factory, mockPendingCache)
	ctx := context.Background()

	job := &river.Job[ProcessAttachmentsArgs]{
		Args: ProcessAttachmentsArgs{
			EmailID: "email_cache_fail",
			UserID:  "user_1",
			Attachments: []AttachmentSource{
				{Filename: "test.txt", ContentType: "text/plain", Base64Content: "SGVsbG8="},
			},
		},
	}

	// Expect S3 upload success
	mockS3Factory.EXPECT().Bucket(s3.BucketAttachments).Return(mockS3Client).AnyTimes()
	mockS3Client.EXPECT().UploadAttachment(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	// Expect cache storage failure
	mockPendingCache.EXPECT().Store(ctx, gomock.Any()).Return(assert.AnError)

	// Expect cleanup of uploaded attachment
	mockS3Client.EXPECT().DeleteObject(ctx, gomock.Any()).Return(nil)

	err := worker.Work(ctx, job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store pending data")
}
