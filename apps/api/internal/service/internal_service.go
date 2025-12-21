package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository/postgres"
	"github.com/emailapi/api/internal/worker"
)

// InternalService handles internal webhook endpoints.
type InternalService struct {
	emailRepo   *postgres.EmailRepository
	riverClient *river.Client[pgx.Tx]
}

// NewInternalService creates a new InternalService.
func NewInternalService(emailRepo *postgres.EmailRepository, riverClient *river.Client[pgx.Tx]) *InternalService {
	return &InternalService{
		emailRepo:   emailRepo,
		riverClient: riverClient,
	}
}

// GuardDutyScanResult represents the scan result from GuardDuty via EventBridge.
type GuardDutyScanResult struct {
	S3Bucket   string `json:"s3_bucket"`
	S3Key      string `json:"s3_key"`
	ScanStatus string `json:"scan_status"` // CLEAN, THREATS_FOUND, UNSUPPORTED, FAILED
	ThreatName string `json:"threat_name"`
}

// HandleGuardDutyScanResult processes scan results from GuardDuty.
func (s *InternalService) HandleGuardDutyScanResult(ctx context.Context, result *GuardDutyScanResult) error {
	// 1. Find attachment by S3 key
	attachment, err := s.emailRepo.GetAttachmentByS3Key(ctx, result.S3Key)
	if err != nil {
		return fmt.Errorf("attachment not found for S3 key %s: %w", result.S3Key, err)
	}

	// 2. Map GuardDuty status to our scan status
	var scanStatus domain.AttachmentScanStatus
	switch result.ScanStatus {
	case "CLEAN":
		scanStatus = domain.ScanStatusClean
	case "THREATS_FOUND":
		scanStatus = domain.ScanStatusThreatsFound
	case "UNSUPPORTED":
		scanStatus = domain.ScanStatusUnsupported
	case "FAILED":
		scanStatus = domain.ScanStatusFailed
	default:
		scanStatus = domain.ScanStatusPending
	}

	// 3. Update attachment scan status
	if err := s.emailRepo.UpdateAttachmentScanStatus(ctx, attachment.ID, scanStatus); err != nil {
		return fmt.Errorf("failed to update attachment scan status: %w", err)
	}

	// 4. Check if all attachments for this email are scanned
	allScanned, allClean, hasThreats, err := s.emailRepo.CheckAllAttachmentsScanned(ctx, attachment.EmailID)
	if err != nil {
		return fmt.Errorf("failed to check attachment scan status: %w", err)
	}

	if !allScanned {
		// Still waiting for other attachments to be scanned
		return nil
	}

	if hasThreats {
		// At least one attachment has threats - mark email as scan_failed
		threatMsg := "Malware detected in attachment"
		if result.ThreatName != "" {
			threatMsg = fmt.Sprintf("Malware detected: %s", result.ThreatName)
		}
		if err := s.emailRepo.UpdateStatus(ctx, attachment.EmailID, domain.EmailStatusScanFailed, threatMsg); err != nil {
			return fmt.Errorf("failed to update email status: %w", err)
		}
		return nil
	}

	if allClean {
		// All attachments are clean - enqueue send email job
		_, err := s.riverClient.Insert(ctx, worker.SendEmailArgs{EmailID: attachment.EmailID}, nil)
		if err != nil {
			return fmt.Errorf("failed to enqueue send email job: %w", err)
		}

		// Update status to queued
		if err := s.emailRepo.UpdateStatus(ctx, attachment.EmailID, domain.EmailStatusQueued, ""); err != nil {
			return fmt.Errorf("failed to update email status: %w", err)
		}
	}

	return nil
}
