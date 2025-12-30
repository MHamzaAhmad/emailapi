package worker

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/emailapi/api/internal/external/s3"
)

// ProcessAttachmentsArgs contains email data + attachments to process.
// After processing, this job enqueues a SendEmailArgs job.
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
	// DryRun mode skips S3 upload and returns fake keys (for performance testing)
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
type AttachmentWorker struct {
	river.WorkerDefaults[ProcessAttachmentsArgs]
	s3Factory   *s3.Factory
	riverClient RiverClient
}

// RiverClient interface for enqueueing jobs.
type RiverClient interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// NewAttachmentWorker creates a new AttachmentWorker.
func NewAttachmentWorker(s3Factory *s3.Factory, riverClient RiverClient) *AttachmentWorker {
	return &AttachmentWorker{
		s3Factory:   s3Factory,
		riverClient: riverClient,
	}
}

// Work processes attachments: downloads from URL or decodes base64, uploads to S3, then enqueues send job.
func (w *AttachmentWorker) Work(ctx context.Context, job *river.Job[ProcessAttachmentsArgs]) error {
	args := job.Args

	// Handle dry-run mode: skip S3 uploads, generate fake keys
	if args.DryRun {
		// Create fake attachment keys for dry-run
		fakeKeys := make([]AttachmentInfo, len(args.Attachments))
		for i, att := range args.Attachments {
			fakeKeys[i] = AttachmentInfo{
				S3Key:       fmt.Sprintf("dry-run/%s/%s", args.EmailID, att.Filename),
				Filename:    att.Filename,
				ContentType: att.ContentType,
			}
		}

		// Enqueue the send email job with dry-run flag
		sendArgs := SendEmailArgs{
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
			AttachmentKeys:         fakeKeys,
			ScheduledAt:            args.ScheduledAt,
			DryRun:                 true, // Propagate dry-run flag
			UnsubscribeBaseURL:     args.UnsubscribeBaseURL,
			UnsubscribeTokenSecret: args.UnsubscribeTokenSecret,
		}

		insertOpts := &river.InsertOpts{}
		if args.ScheduledAt != nil {
			insertOpts.ScheduledAt = *args.ScheduledAt
		}

		if _, err := w.riverClient.Insert(ctx, sendArgs, insertOpts); err != nil {
			return fmt.Errorf("failed to enqueue send job: %w", err)
		}

		fmt.Printf("[DRY-RUN] Simulated processing %d attachments for email %s\n", len(args.Attachments), args.EmailID)
		return nil
	}

	// Process each attachment in parallel and upload to S3
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
			return err
		}
	}

	// Enqueue the send email job with attachment info
	sendArgs := SendEmailArgs{
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
		AttachmentKeys:         attachmentKeys,
		ScheduledAt:            args.ScheduledAt,
		DryRun:                 false, // Normal mode
		UnsubscribeBaseURL:     args.UnsubscribeBaseURL,
		UnsubscribeTokenSecret: args.UnsubscribeTokenSecret,
	}

	// Prepare insert options
	insertOpts := &river.InsertOpts{}
	if args.ScheduledAt != nil {
		insertOpts.ScheduledAt = *args.ScheduledAt
	}

	if _, err := w.riverClient.Insert(ctx, sendArgs, insertOpts); err != nil {
		// Clean up uploaded attachments on failure
		for _, att := range attachmentKeys {
			w.s3Factory.Bucket(s3.BucketAttachments).DeleteObject(ctx, att.S3Key)
		}
		return fmt.Errorf("failed to enqueue send job: %w", err)
	}

	fmt.Printf("Processed %d attachments for email %s, enqueued for sending\n", len(attachmentKeys), args.EmailID)
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
