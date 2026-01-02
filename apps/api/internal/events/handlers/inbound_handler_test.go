package handlers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/emailapi/api/internal/events/handlers"
	"github.com/emailapi/api/internal/events/handlers/mocks"
	s3Mocks "github.com/emailapi/api/internal/external/s3/mocks"
)

func TestInboundEmailHandler_HandleS3Event(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockS3Factory := s3Mocks.NewMockFactoryInterface(ctrl)
	mockProcessor := mocks.NewMockInboundEmailProcessor(ctrl)

	ctx := context.Background()
	bucket := "test-bucket"
	key := "emails/123/raw"

	t.Run("success", func(t *testing.T) {
		handler := handlers.NewInboundEmailHandler(mockS3Factory, mockProcessor)

		mockProcessor.EXPECT().
			ProcessRawEmail(ctx, bucket, key).
			Return(nil)

		err := handler.HandleS3Event(ctx, bucket, key)
		require.NoError(t, err)
	})

	t.Run("no processor configured", func(t *testing.T) {
		handler := handlers.NewInboundEmailHandler(mockS3Factory, nil)

		err := handler.HandleS3Event(ctx, bucket, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no processor configured")
	})

	t.Run("processor error", func(t *testing.T) {
		handler := handlers.NewInboundEmailHandler(mockS3Factory, mockProcessor)

		mockProcessor.EXPECT().
			ProcessRawEmail(ctx, bucket, key).
			Return(errors.New("failed to download from S3"))

		err := handler.HandleS3Event(ctx, bucket, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to process inbound email")
	})
}
