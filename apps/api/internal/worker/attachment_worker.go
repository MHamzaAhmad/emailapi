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

// GuardDuty scan status constants
const (
	// GuardDutyTagKey is the S3 object tag key used by GuardDuty Malware Protection
	GuardDutyTagKey = "GuardDutyMalwareScanStatus"

	// GuardDuty scan result values
	ScanStatusClean        = "NO_THREATS_FOUND"
	ScanStatusThreat       = "THREATS_FOUND"
	ScanStatusUnsupported  = "UNSUPPORTED"
	ScanStatusAccessDenied = "ACCESS_DENIED"
	ScanStatusFailed       = "FAILED"

	// Scan wait configuration
	MaxScanAttempts = 10               // 10 attempts × 30s = 5 min max wait
	ScanSnoozeTime  = 30 * time.Second // Time between scan status checks
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
	// UploadedKeys stores S3 keys of uploaded attachments (populated after first attempt)
	// This field persists across job snooze cycles for GuardDuty scan polling
	UploadedKeys []AttachmentInfo `json:"uploaded_keys,omitempty"`
	// ScanAttemptCount tracks how many times we've checked for scan results
	ScanAttemptCount int `json:"scan_attempt_count,omitempty"`
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
	s3Factory   s3.FactoryInterface
	riverClient RiverClient
}

// RiverClient interface for enqueueing jobs.
//
//go:generate mockgen -destination=mocks/mock_river_client.go -package=mocks . RiverClient
type RiverClient interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// NewAttachmentWorker creates a new AttachmentWorker.
func NewAttachmentWorker(s3Factory s3.FactoryInterface, riverClient RiverClient) *AttachmentWorker {
	return &AttachmentWorker{
		s3Factory:   s3Factory,
		riverClient: riverClient,
	}
}

// Work processes attachments: downloads from URL or decodes base64, uploads to S3,
// waits for GuardDuty malware scan to complete, then enqueues send job if clean.
func (w *AttachmentWorker) Work(ctx context.Context, job *river.Job[ProcessAttachmentsArgs]) error {
	args := job.Args

	// Handle dry-run mode: skip S3 uploads and scanning, generate fake keys
	if args.DryRun {
		return w.handleDryRun(ctx, &args)
	}

	// First attempt: upload attachments to S3
	if len(args.UploadedKeys) == 0 {
		attachmentKeys, err := w.uploadAttachments(ctx, &args)
		if err != nil {
			return err
		}
		// Store keys in job args for subsequent attempts (River preserves args across snooze)
		args.UploadedKeys = attachmentKeys

		// Snooze to wait for GuardDuty scan
		fmt.Printf("Uploaded %d attachments for email %s, waiting for GuardDuty scan...\n", len(attachmentKeys), args.EmailID)
		return river.JobSnooze(ScanSnoozeTime)
	}

	// Subsequent attempts: check scan status
	allClean, hasThreat, pendingCount, err := w.checkScanStatus(ctx, args.UploadedKeys)
	if err != nil {
		return err
	}

	// Threat detected - abort and cleanup
	if hasThreat {
		fmt.Printf("Malware detected in attachment for email %s - rejecting\n", args.EmailID)
		w.cleanupAttachments(ctx, args.UploadedKeys)
		return fmt.Errorf("attachment contains malware - email rejected")
	}

	// All attachments scanned clean - proceed to send
	if allClean {
		fmt.Printf("All %d attachments clean for email %s, enqueuing send job\n", len(args.UploadedKeys), args.EmailID)
		return w.enqueueSendJob(ctx, &args)
	}

	// Still pending - check timeout
	args.ScanAttemptCount++
	if args.ScanAttemptCount >= MaxScanAttempts {
		fmt.Printf("Scan timeout for email %s: %d attachments not scanned within %v\n", args.EmailID, pendingCount, time.Duration(MaxScanAttempts)*ScanSnoozeTime)
		w.cleanupAttachments(ctx, args.UploadedKeys)
		return fmt.Errorf("scan timeout: %d attachments not scanned within %v", pendingCount, time.Duration(MaxScanAttempts)*ScanSnoozeTime)
	}

	// Keep waiting
	fmt.Printf("Waiting for GuardDuty scan (attempt %d/%d) for email %s, %d pending\n", args.ScanAttemptCount, MaxScanAttempts, args.EmailID, pendingCount)
	return river.JobSnooze(ScanSnoozeTime)
}

// handleDryRun processes attachments in dry-run mode without S3 uploads or scanning
func (w *AttachmentWorker) handleDryRun(ctx context.Context, args *ProcessAttachmentsArgs) error {
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
		DryRun:                 true,
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

// checkScanStatus polls S3 tags for all attachments to check GuardDuty scan results
func (w *AttachmentWorker) checkScanStatus(ctx context.Context, keys []AttachmentInfo) (allClean, hasThreat bool, pendingCount int, err error) {
	allClean = true

	for _, att := range keys {
		tags, err := w.s3Factory.Bucket(s3.BucketAttachments).GetObjectTags(ctx, att.S3Key)
		if err != nil {
			return false, false, 0, fmt.Errorf("failed to get tags for %s: %w", att.Filename, err)
		}

		status := tags[GuardDutyTagKey]
		switch status {
		case ScanStatusClean:
			continue // Good
		case ScanStatusThreat:
			return false, true, 0, nil // Early exit on threat
		case ScanStatusUnsupported, ScanStatusFailed, ScanStatusAccessDenied:
			// Policy: allow unsupported files (e.g., encrypted ZIPs) to proceed
			// These can't be scanned but are not necessarily malicious
			continue
		default:
			// No tag yet = still scanning
			allClean = false
			pendingCount++
		}
	}

	return allClean, false, pendingCount, nil
}

// cleanupAttachments removes all uploaded attachments from S3
func (w *AttachmentWorker) cleanupAttachments(ctx context.Context, keys []AttachmentInfo) {
	for _, att := range keys {
		if att.S3Key != "" {
			_ = w.s3Factory.Bucket(s3.BucketAttachments).DeleteObject(ctx, att.S3Key)
		}
	}
}

// enqueueSendJob creates and enqueues the SendEmailArgs job
func (w *AttachmentWorker) enqueueSendJob(ctx context.Context, args *ProcessAttachmentsArgs) error {
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
		AttachmentKeys:         args.UploadedKeys,
		ScheduledAt:            args.ScheduledAt,
		DryRun:                 false,
		UnsubscribeBaseURL:     args.UnsubscribeBaseURL,
		UnsubscribeTokenSecret: args.UnsubscribeTokenSecret,
	}

	insertOpts := &river.InsertOpts{}
	if args.ScheduledAt != nil {
		insertOpts.ScheduledAt = *args.ScheduledAt
	}

	if _, err := w.riverClient.Insert(ctx, sendArgs, insertOpts); err != nil {
		// Clean up uploaded attachments on failure
		w.cleanupAttachments(ctx, args.UploadedKeys)
		return fmt.Errorf("failed to enqueue send job: %w", err)
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
