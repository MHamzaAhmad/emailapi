package service

import (
	"context"
)

// InternalService handles internal webhook endpoints.
// Note: With the stateless email architecture, GuardDuty scan results are handled
// in the attachment worker job flow rather than via webhooks to PG.
type InternalService struct{}

// NewInternalService creates a new InternalService.
func NewInternalService() *InternalService {
	return &InternalService{}
}

// GuardDutyScanResult represents the scan result from GuardDuty via EventBridge.
type GuardDutyScanResult struct {
	S3Bucket   string `json:"s3_bucket"`
	S3Key      string `json:"s3_key"`
	ScanStatus string `json:"scan_status"` // CLEAN, THREATS_FOUND, UNSUPPORTED, FAILED
	ThreatName string `json:"threat_name"`
}

// HandleGuardDutyScanResult is a placeholder for scan result handling.
// In the stateless architecture, scan results are handled inline during attachment processing.
// This endpoint can be used for logging/monitoring if needed.
func (s *InternalService) HandleGuardDutyScanResult(ctx context.Context, result *GuardDutyScanResult) error {
	// With the new stateless architecture, attachment scanning is handled
	// in the attachment worker before enqueuing the send job.
	// This endpoint is kept for backward compatibility but does nothing.
	return nil
}
