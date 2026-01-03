package worker

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/external/s3"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
)

// ProcessAttachmentsArgs contains email data + attachments to process.
// After uploading, pending data is stored in Redis for the GuardDuty handler.
type ProcessAttachmentsArgs struct {
	// Full email data (embedded, not stored)
	EmailID    string            `json:"email_id"`
	UserID     string            `json:"user_id"`
	From       string            `json:"from"`
	To         []string          `json:"to"`
	Cc         []string          `json:"cc,omitempty"`
	Bcc        []string          `json:"bcc,omitempty"`
	Subject    string            `json:"subject"`
	Body       string            `json:"body,omitempty"`
	HTML       string            `json:"html,omitempty"`
	InReplyTo  string            `json:"in_reply_to,omitempty"`
	References []string          `json:"references,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	// Attachments to download/decode and upload to S3
	Attachments []AttachmentSource `json:"attachments"`
	// Optional scheduled time for email delivery
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	// DryRun mode skips S3 upload and scanning, directly enqueues send job
	DryRun bool `json:"dry_run,omitempty"`
	// Unsubscribe config for worker splitting (passthrough to SendEmailArgs)
	UnsubscribeBaseURL     string `json:"unsubscribe_base_url,omitempty"`
	UnsubscribeTokenSecret string `json:"unsubscribe_token_secret,omitempty"`
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
// Uses event-driven GuardDuty integration - uploads to S3, stores pending data in Redis,
// then GuardDuty handler picks up from EventBridge events.
type AttachmentWorker struct {
	river.WorkerDefaults[ProcessAttachmentsArgs]
	s3Factory    s3.FactoryInterface
	pendingCache redisrepo.PendingAttachmentCacheInterface
}

// NewAttachmentWorker creates a new AttachmentWorker.
func NewAttachmentWorker(
	s3Factory s3.FactoryInterface,
	pendingCache redisrepo.PendingAttachmentCacheInterface,
) *AttachmentWorker {
	return &AttachmentWorker{
		s3Factory:    s3Factory,
		pendingCache: pendingCache,
	}
}

// Work processes attachments: downloads from URL or decodes base64, uploads to S3,
// stores pending email data in Redis for GuardDuty event handler.
func (w *AttachmentWorker) Work(ctx context.Context, job *river.Job[ProcessAttachmentsArgs]) error {
	args := &job.Args

	// Handle dry-run mode: skip S3 uploads and scanning
	if args.DryRun {
		return w.handleDryRun(ctx, args)
	}

	// Upload attachments to S3
	attachmentKeys, err := w.uploadAttachments(ctx, args)
	if err != nil {
		return err
	}

	// Store pending email data in Redis for GuardDuty handler to pick up
	if err := w.storePendingData(ctx, args, attachmentKeys); err != nil {
		// Clean up uploaded attachments on cache failure
		w.cleanupAttachments(ctx, attachmentKeys)
		return fmt.Errorf("failed to store pending data: %w", err)
	}

	fmt.Printf("Uploaded %d attachments for email %s, waiting for GuardDuty scan events...\n", len(attachmentKeys), args.EmailID)
	return nil
}

// storePendingData stores email data in Redis for the GuardDuty handler to resume
func (w *AttachmentWorker) storePendingData(ctx context.Context, args *ProcessAttachmentsArgs, keys []AttachmentInfo) error {
	// Convert to cache format
	cacheKeys := make([]redisrepo.AttachmentKeyInfo, len(keys))
	for i, k := range keys {
		cacheKeys[i] = redisrepo.AttachmentKeyInfo{
			S3Key:       k.S3Key,
			Filename:    k.Filename,
			ContentType: k.ContentType,
		}
	}

	data := &redisrepo.PendingAttachmentData{
		EmailID:                args.EmailID,
		UserID:                 args.UserID,
		From:                   args.From,
		To:                     args.To,
		Cc:                     args.Cc,
		Bcc:                    args.Bcc,
		Subject:                args.Subject,
		Body:                   args.Body,
		HTML:                   args.HTML,
		InReplyTo:              args.InReplyTo,
		References:             args.References,
		Metadata:               args.Metadata,
		ScheduledAt:            args.ScheduledAt,
		UnsubscribeBaseURL:     args.UnsubscribeBaseURL,
		UnsubscribeTokenSecret: args.UnsubscribeTokenSecret,
		AttachmentKeys:         cacheKeys,
		PendingCount:           len(keys),
	}

	return w.pendingCache.Store(ctx, data)
}

// handleDryRun processes attachments in dry-run mode without S3 uploads or scanning
func (w *AttachmentWorker) handleDryRun(ctx context.Context, args *ProcessAttachmentsArgs) error {
	fmt.Printf("[DRY-RUN] Simulated processing %d attachments for email %s (event-driven mode)\n", len(args.Attachments), args.EmailID)
	// In dry-run mode, we just log - no actual processing
	// The test UI should handle this case separately
	return nil
}

// uploadAttachments downloads/decodes and uploads all attachments to S3 in parallel
func (w *AttachmentWorker) uploadAttachments(ctx context.Context, args *ProcessAttachmentsArgs) ([]AttachmentInfo, error) {
	attachmentKeys := make([]AttachmentInfo, len(args.Attachments))
	errs := make([]error, len(args.Attachments))

	var wg sync.WaitGroup
	for i, att := range args.Attachments {
		wg.Add(1)
		go func(idx int, att AttachmentSource) {
			defer wg.Done()

			content, contentType, err := w.getAttachmentContent(ctx, att)
			if err != nil {
				errs[idx] = fmt.Errorf("failed to get attachment content for %s: %w", att.Filename, err)
				return
			}

			// Generate S3 key
			s3Key := fmt.Sprintf("attachments/%s/%s/%s", args.EmailID, uuid.New().String(), att.Filename)

			// Upload to S3
			if err := w.s3Factory.Bucket(s3.BucketAttachments).UploadAttachment(ctx, s3Key, content, contentType); err != nil {
				errs[idx] = fmt.Errorf("failed to upload attachment %s: %w", att.Filename, err)
				return
			}

			attachmentKeys[idx] = AttachmentInfo{
				S3Key:       s3Key,
				Filename:    att.Filename,
				ContentType: contentType,
			}
		}(i, att)
	}
	wg.Wait()

	// Check for any errors during parallel processing
	for _, err := range errs {
		if err != nil {
			// Clean up any successfully uploaded attachments
			for _, att := range attachmentKeys {
				if att.S3Key != "" {
					w.s3Factory.Bucket(s3.BucketAttachments).DeleteObject(ctx, att.S3Key)
				}
			}
			return nil, err
		}
	}

	return attachmentKeys, nil
}

// cleanupAttachments removes all uploaded attachments from S3
func (w *AttachmentWorker) cleanupAttachments(ctx context.Context, keys []AttachmentInfo) {
	for _, att := range keys {
		if att.S3Key != "" {
			_ = w.s3Factory.Bucket(s3.BucketAttachments).DeleteObject(ctx, att.S3Key)
		}
	}
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
