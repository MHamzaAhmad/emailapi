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

	t.Run("first attempt uploads and snoozes", func(t *testing.T) {
		job := &river.Job[ProcessAttachmentsArgs]{
			Args: ProcessAttachmentsArgs{
				EmailID: "email_real",
				UserID:  "user_1",
				Attachments: []AttachmentSource{
					{Filename: "test.txt", ContentType: "text/plain", Base64Content: "SGVsbG8="},
				},
				// No UploadedKeys = first attempt
			},
		}

		// Expect S3 bucket retrieval and upload
		mockS3Factory.EXPECT().Bucket(s3.BucketAttachments).Return(mockS3Client).AnyTimes()
		mockS3Client.EXPECT().UploadAttachment(ctx, gomock.Any(), []byte("Hello"), "text/plain").Return(nil)

		err := worker.Work(ctx, job)
		// Should return JobSnooze error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "JobSnoozeError")
	})

	t.Run("subsequent attempt with clean scan enqueues send", func(t *testing.T) {
		job := &river.Job[ProcessAttachmentsArgs]{
			Args: ProcessAttachmentsArgs{
				EmailID: "email_real",
				UserID:  "user_1",
				Attachments: []AttachmentSource{
					{Filename: "test.txt", ContentType: "text/plain", Base64Content: "SGVsbG8="},
				},
				// UploadedKeys present = subsequent attempt
				UploadedKeys: []AttachmentInfo{
					{S3Key: "attachments/email_real/uuid/test.txt", Filename: "test.txt", ContentType: "text/plain"},
				},
			},
		}

		// Expect S3 GetObjectTags to return clean status
		mockS3Factory.EXPECT().Bucket(s3.BucketAttachments).Return(mockS3Client).AnyTimes()
		mockS3Client.EXPECT().GetObjectTags(ctx, "attachments/email_real/uuid/test.txt").Return(map[string]string{
			GuardDutyTagKey: ScanStatusClean,
		}, nil)

		// Expect River Insert with real keys
		mockRiver.EXPECT().Insert(ctx, gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
			sendArgs, ok := args.(SendEmailArgs)
			require.True(t, ok)
			assert.False(t, sendArgs.DryRun)
			assert.Equal(t, "email_real", sendArgs.EmailID)
			assert.Len(t, sendArgs.AttachmentKeys, 1)
			return &rivertype.JobInsertResult{}, nil
		})

		err := worker.Work(ctx, job)
		require.NoError(t, err)
	})
}

func TestAttachmentWorker_Work_ThreatDetected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockS3Client := s3Mocks.NewMockClient(ctrl)
	mockRiver := mocks.NewMockRiverClient(ctrl)

	worker := NewAttachmentWorker(mockS3Factory, mockRiver)
	ctx := context.Background()

	job := &river.Job[ProcessAttachmentsArgs]{
		Args: ProcessAttachmentsArgs{
			EmailID: "email_threat",
			UserID:  "user_1",
			UploadedKeys: []AttachmentInfo{
				{S3Key: "attachments/email_threat/uuid/malware.exe", Filename: "malware.exe", ContentType: "application/octet-stream"},
			},
		},
	}

	// Expect S3 GetObjectTags to return threat status
	mockS3Factory.EXPECT().Bucket(s3.BucketAttachments).Return(mockS3Client).AnyTimes()
	mockS3Client.EXPECT().GetObjectTags(ctx, "attachments/email_threat/uuid/malware.exe").Return(map[string]string{
		GuardDutyTagKey: ScanStatusThreat,
	}, nil)

	// Expect cleanup
	mockS3Client.EXPECT().DeleteObject(ctx, "attachments/email_threat/uuid/malware.exe").Return(nil)

	err := worker.Work(ctx, job)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malware")
}

func TestAttachmentWorker_Work_ScanPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockS3Client := s3Mocks.NewMockClient(ctrl)
	mockRiver := mocks.NewMockRiverClient(ctrl)

	worker := NewAttachmentWorker(mockS3Factory, mockRiver)
	ctx := context.Background()

	job := &river.Job[ProcessAttachmentsArgs]{
		Args: ProcessAttachmentsArgs{
			EmailID: "email_pending",
			UserID:  "user_1",
			UploadedKeys: []AttachmentInfo{
				{S3Key: "attachments/email_pending/uuid/file.pdf", Filename: "file.pdf", ContentType: "application/pdf"},
			},
		},
	}

	// Expect S3 GetObjectTags to return no tag (still scanning)
	mockS3Factory.EXPECT().Bucket(s3.BucketAttachments).Return(mockS3Client).AnyTimes()
	mockS3Client.EXPECT().GetObjectTags(ctx, "attachments/email_pending/uuid/file.pdf").Return(map[string]string{}, nil)

	err := worker.Work(ctx, job)
	// Should return JobSnooze error (still waiting)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JobSnoozeError")
}
