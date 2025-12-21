package worker

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/repository/postgres"
)

// ProcessAttachmentsArgs defines the arguments for the process_attachments job.
type ProcessAttachmentsArgs struct {
	EmailID     string             `json:"email_id"`
	Attachments []AttachmentSource `json:"attachments"`
}

// AttachmentSource represents an attachment to process (from request).
type AttachmentSource struct {
	Filename      string `json:"filename"`
	ContentType   string `json:"content_type"`
	URL           string `json:"url,omitempty"`
	Base64Content string `json:"base64_content,omitempty"`
}

func (ProcessAttachmentsArgs) Kind() string { return "process_attachments" }

// AttachmentWorker handles attachment processing jobs.
type AttachmentWorker struct {
	river.WorkerDefaults[ProcessAttachmentsArgs]
	s3Client  s3.Client
	emailRepo *postgres.EmailRepository
}

// NewAttachmentWorker creates a new AttachmentWorker.
func NewAttachmentWorker(s3Client s3.Client, emailRepo *postgres.EmailRepository) *AttachmentWorker {
	return &AttachmentWorker{
		s3Client:  s3Client,
		emailRepo: emailRepo,
	}
}

// Work processes attachments: downloads from URL or decodes base64, uploads to S3.
func (w *AttachmentWorker) Work(ctx context.Context, job *river.Job[ProcessAttachmentsArgs]) error {
	emailID := job.Args.EmailID

	// Update email status to processing_attachments
	if err := w.emailRepo.UpdateStatus(ctx, emailID, domain.EmailStatusProcessingAttachments, ""); err != nil {
		return fmt.Errorf("failed to update email status: %w", err)
	}

	// Process each attachment
	for _, att := range job.Args.Attachments {
		content, contentType, err := w.getAttachmentContent(ctx, att)
		if err != nil {
			// Update email status to failed
			w.emailRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, fmt.Sprintf("Failed to process attachment %s: %v", att.Filename, err))
			return fmt.Errorf("failed to get attachment content for %s: %w", att.Filename, err)
		}

		// Generate S3 key
		s3Key := fmt.Sprintf("attachments/%s/%s/%s", emailID, uuid.New().String(), att.Filename)

		// Upload to S3
		if err := w.s3Client.UploadAttachment(ctx, s3Key, content, contentType); err != nil {
			w.emailRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, fmt.Sprintf("Failed to upload attachment %s: %v", att.Filename, err))
			return fmt.Errorf("failed to upload attachment %s: %w", att.Filename, err)
		}

		// Get object metadata for size
		meta, err := w.s3Client.HeadObject(ctx, s3Key)
		if err != nil {
			// Non-fatal, just use content length
			meta = &s3.ObjectMeta{SizeBytes: int64(len(content)), ContentType: contentType}
		}

		// Create attachment record in database
		attachment := &domain.EmailAttachment{
			ID:          uuid.New().String(),
			EmailID:     emailID,
			Filename:    att.Filename,
			ContentType: contentType,
			S3Key:       s3Key,
			SizeBytes:   meta.SizeBytes,
			ScanStatus:  domain.ScanStatusPending,
		}

		if err := w.emailRepo.CreateAttachment(ctx, attachment); err != nil {
			w.emailRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, fmt.Sprintf("Failed to save attachment record %s: %v", att.Filename, err))
			return fmt.Errorf("failed to create attachment record: %w", err)
		}
	}

	// Update email status to scanning_attachments
	// GuardDuty will scan the files and EventBridge will call our webhook
	if err := w.emailRepo.UpdateStatus(ctx, emailID, domain.EmailStatusScanningAttachments, ""); err != nil {
		return fmt.Errorf("failed to update email status to scanning: %w", err)
	}

	return nil
}

// getAttachmentContent retrieves attachment content from URL or decodes base64.
func (w *AttachmentWorker) getAttachmentContent(ctx context.Context, att AttachmentSource) ([]byte, string, error) {
	contentType := att.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if att.URL != "" {
		// Download from URL
		content, detectedType, err := s3.DownloadFromURL(ctx, att.URL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to download from URL: %w", err)
		}
		// Prefer detected content type if not explicitly set
		if att.ContentType == "" && detectedType != "" {
			contentType = detectedType
		}
		return content, contentType, nil
	}

	if att.Base64Content != "" {
		// Decode base64
		content, err := base64.StdEncoding.DecodeString(att.Base64Content)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode base64 content: %w", err)
		}
		return content, contentType, nil
	}

	return nil, "", fmt.Errorf("attachment has neither URL nor base64 content")
}
