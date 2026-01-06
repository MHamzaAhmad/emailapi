package handlers

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/emailapi/api/internal/events"
	"github.com/emailapi/api/internal/external/s3"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/worker"
)

// GuardDutyHandler processes GuardDuty Malware Protection scan result events.
type GuardDutyHandler struct {
	pendingCache redisrepo.PendingAttachmentCacheInterface
	riverClient  RiverClient
	s3Factory    s3.FactoryInterface
}

// RiverClient interface for enqueueing jobs.
type RiverClient interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// NewGuardDutyHandler creates a new GuardDuty scan result handler.
func NewGuardDutyHandler(
	pendingCache redisrepo.PendingAttachmentCacheInterface,
	riverClient RiverClient,
	s3Factory s3.FactoryInterface,
) *GuardDutyHandler {
	return &GuardDutyHandler{
		pendingCache: pendingCache,
		riverClient:  riverClient,
		s3Factory:    s3Factory,
	}
}

// Handle processes a GuardDuty scan result event.
func (h *GuardDutyHandler) Handle(ctx context.Context, detail *events.GuardDutyEventDetail) error {
	s3Key := detail.S3ObjectDetails.ObjectKey
	scanResult := detail.ScanResultDetails.ScanResult

	fmt.Printf("GuardDuty scan result for %s: %s\n", s3Key, scanResult)

	// Mark this attachment as scanned and check if all attachments for the email are done
	data, allScanned, err := h.pendingCache.MarkAttachmentScanned(ctx, s3Key)
	if err != nil {
		return fmt.Errorf("failed to mark attachment scanned: %w", err)
	}
	if data == nil {
		// Not found - could be expired, already processed, or not our attachment
		fmt.Printf("No pending email found for S3 key %s (may be expired or already processed)\n", s3Key)
		return nil
	}

	// Handle threat detection - abort all attachments for this email
	if scanResult == events.GuardDutyScanResultThreat {
		fmt.Printf("Malware detected in attachment for email %s - rejecting\n", data.EmailID)
		h.cleanupEmail(ctx, data)
		return nil // Don't return error - we've handled it
	}

	// If scan failed or access denied, we could choose to proceed or abort
	// Policy: proceed for unsupported/failed, user can decide
	if scanResult == events.GuardDutyScanResultFailed || scanResult == events.GuardDutyScanResultAccessDenied {
		fmt.Printf("Scan %s for %s - proceeding anyway\n", scanResult, s3Key)
	}

	// Check if all attachments are scanned
	if !allScanned {
		fmt.Printf("Email %s: waiting for %d more attachments to scan\n", data.EmailID, data.PendingCount)
		return nil
	}

	// All attachments scanned clean - enqueue send job
	fmt.Printf("All attachments clean for email %s, enqueuing send job\n", data.EmailID)
	return h.enqueueSendJob(ctx, data)
}

// enqueueSendJob creates and enqueues the SendEmailArgs job.
func (h *GuardDutyHandler) enqueueSendJob(ctx context.Context, data *redisrepo.PendingAttachmentData) error {
	// Convert attachment keys to worker format
	attachmentKeys := make([]worker.AttachmentInfo, len(data.AttachmentKeys))
	for i, att := range data.AttachmentKeys {
		attachmentKeys[i] = worker.AttachmentInfo{
			S3Key:       att.S3Key,
			Filename:    att.Filename,
			ContentType: att.ContentType,
		}
	}

	sendArgs := worker.SendEmailArgs{
		EmailID:                data.EmailID,
		UserID:                 data.UserID,
		From:                   data.From,
		To:                     data.To,
		Cc:                     data.Cc,
		Bcc:                    data.Bcc,
		Subject:                data.Subject,
		Body:                   data.Body,
		HTML:                   data.HTML,
		InReplyTo:              data.InReplyTo,
		References:             data.References,
		Metadata:               data.Metadata,
		AttachmentKeys:         attachmentKeys,
		ScheduledAt:            data.ScheduledAt,
		UnsubscribeBaseURL:     data.UnsubscribeBaseURL,
		UnsubscribeTokenSecret: data.UnsubscribeTokenSecret,
	}

	insertOpts := &river.InsertOpts{}
	if data.ScheduledAt != nil {
		insertOpts.ScheduledAt = *data.ScheduledAt
	}

	if _, err := h.riverClient.Insert(ctx, sendArgs, insertOpts); err != nil {
		return fmt.Errorf("failed to enqueue send job: %w", err)
	}

	// Clean up pending data from cache
	if err := h.pendingCache.Delete(ctx, data.EmailID); err != nil {
		fmt.Printf("Warning: failed to delete pending data for %s: %v\n", data.EmailID, err)
	}

	return nil
}

// cleanupEmail removes pending data and S3 objects for a rejected email.
func (h *GuardDutyHandler) cleanupEmail(ctx context.Context, data *redisrepo.PendingAttachmentData) {
	// Delete all attachments from S3
	for _, att := range data.AttachmentKeys {
		if err := h.s3Factory.Bucket(s3.BucketAttachments).DeleteObject(ctx, att.S3Key); err != nil {
			fmt.Printf("Warning: failed to delete attachment %s: %v\n", att.S3Key, err)
		}
	}

	// Remove pending data from cache
	if err := h.pendingCache.Delete(ctx, data.EmailID); err != nil {
		fmt.Printf("Warning: failed to delete pending data for %s: %v\n", data.EmailID, err)
	}

	// TODO: Send webhook notification about rejected email
}
