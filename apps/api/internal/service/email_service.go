package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/emailapi/api/gen/emailapi/v1"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/repository/clickhouse"
	"github.com/emailapi/api/internal/worker"
	"github.com/riverqueue/river"
)

type EmailService struct {
	emailapi.UnimplementedEmailServiceServer
	riverClient *river.Client[any]
	s3Client    s3.Client
	chRepo      *clickhouse.EmailRepository
}

func NewEmailService(riverClient *river.Client[any], s3Client s3.Client, chRepo *clickhouse.EmailRepository) *EmailService {
	return &EmailService{
		riverClient: riverClient,
		s3Client:    s3Client,
		chRepo:      chRepo,
	}
}

func (s *EmailService) SendEmail(ctx context.Context, req *emailapi.SendEmailRequest) (*emailapi.SendEmailResponse, error) {
	// 1. Upload attachments to S3
	var attachments []worker.AttachmentMeta
	for _, att := range req.Attachments {
		key := fmt.Sprintf("attachments/%s/%s", uuid.New().String(), att.Filename)
		err := s.s3Client.UploadAttachment(ctx, key, att.Content, att.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to upload attachment %s: %w", att.Filename, err)
		}

		attachments = append(attachments, worker.AttachmentMeta{
			Filename:    att.Filename,
			S3Key:       key,
			ContentType: att.ContentType,
		})
	}

	// 2. Enqueue River Job
	args := worker.SendEmailArgs{
		From:        req.From,
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     req.Subject,
		Body:        req.Body,
		HTML:        req.Html,
		Attachments: attachments,
		Metadata:    req.Metadata,
	}

	insertRes, err := s.riverClient.Insert(ctx, args, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue email job: %w", err)
	}

	// 3. Log to ClickHouse (Optional - "pending" state or "queued")
	// We might leave this to the worker to log "sent" or "failed".

	return &emailapi.SendEmailResponse{
		Id:     fmt.Sprintf("%d", insertRes.Job.ID), // Start with Job ID as Email ID for now? Or use a separate UUID?
		Status: emailapi.EmailStatus_EMAIL_STATUS_PENDING,
	}, nil
}
