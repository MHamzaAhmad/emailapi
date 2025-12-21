package domain

import (
	"time"
)

// EmailStatus represents the current state of an email.
type EmailStatus string

const (
	EmailStatusPending               EmailStatus = "pending"
	EmailStatusProcessingAttachments EmailStatus = "processing_attachments"
	EmailStatusScanningAttachments   EmailStatus = "scanning_attachments"
	EmailStatusScanFailed            EmailStatus = "scan_failed"
	EmailStatusQueued                EmailStatus = "queued"
	EmailStatusSent                  EmailStatus = "sent"
	EmailStatusDelivered             EmailStatus = "delivered"
	EmailStatusFailed                EmailStatus = "failed"
	EmailStatusBounced               EmailStatus = "bounced"
)

// AttachmentScanStatus represents the security scan status of an attachment.
type AttachmentScanStatus string

const (
	ScanStatusPending      AttachmentScanStatus = "pending"
	ScanStatusScanning     AttachmentScanStatus = "scanning"
	ScanStatusClean        AttachmentScanStatus = "clean"
	ScanStatusThreatsFound AttachmentScanStatus = "threats_found"
	ScanStatusFailed       AttachmentScanStatus = "failed"
	ScanStatusUnsupported  AttachmentScanStatus = "unsupported"
)

// Email represents an email entity in the system.
type Email struct {
	ID           string            `json:"id"`
	MessageID    string            `json:"message_id,omitempty"`
	From         string            `json:"from"`
	To           []string          `json:"to"`
	Cc           []string          `json:"cc,omitempty"`
	Bcc          []string          `json:"bcc,omitempty"`
	Subject      string            `json:"subject"`
	Body         string            `json:"body"`
	HTML         string            `json:"html,omitempty"`
	Status       EmailStatus       `json:"status"`
	ProviderID   string            `json:"provider_id,omitempty"`
	UserID       string            `json:"user_id"`
	Metadata     Metadata          `json:"metadata,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Attachments  []EmailAttachment `json:"attachments,omitempty"`
	ScheduledAt  *time.Time        `json:"scheduled_at,omitempty"`
	SentAt       *time.Time        `json:"sent_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// Metadata holds additional key-value data for emails.
type Metadata map[string]interface{}

// EmailAttachment represents an attachment stored in S3 with scan status.
type EmailAttachment struct {
	ID            string               `json:"id"`
	EmailID       string               `json:"email_id"`
	Filename      string               `json:"filename"`
	ContentType   string               `json:"content_type"`
	S3Key         string               `json:"s3_key"`
	SizeBytes     int64                `json:"size_bytes"`
	ScanStatus    AttachmentScanStatus `json:"scan_status"`
	ScanCheckedAt *time.Time           `json:"scan_checked_at,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
}

// Attachment represents an email attachment in a request (before upload).
type Attachment struct {
	Filename      string `json:"filename"`
	ContentType   string `json:"content_type"`
	URL           string `json:"url,omitempty"`
	Base64Content string `json:"base64_content,omitempty"`
}

// SendEmailRequest represents a request to send an email.
type SendEmailRequest struct {
	From        string       `json:"from" binding:"required,email"`
	To          []string     `json:"to" binding:"required,min=1,dive,email"`
	Cc          []string     `json:"cc,omitempty" binding:"omitempty,dive,email"`
	Bcc         []string     `json:"bcc,omitempty" binding:"omitempty,dive,email"`
	Subject     string       `json:"subject" binding:"required"`
	Body        string       `json:"body"`
	HTML        string       `json:"html,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Metadata    Metadata     `json:"metadata,omitempty"`
	ScheduledAt *time.Time   `json:"scheduled_at,omitempty"`
}

// SendEmailResponse represents the response after sending an email.
type SendEmailResponse struct {
	ID     string      `json:"id"`
	Status EmailStatus `json:"status"`
}

// EmailEvent represents an email event for ClickHouse logging.
type EmailEvent struct {
	ID        string    `json:"id"`
	EmailID   string    `json:"email_id"`
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	EventType string    `json:"event_type"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Metadata  string    `json:"metadata"`
	Timestamp time.Time `json:"timestamp"`
}

// EmailStats holds aggregated email statistics.
type EmailStats struct {
	TotalSent      int64 `json:"total_sent"`
	TotalDelivered int64 `json:"total_delivered"`
	TotalBounced   int64 `json:"total_bounced"`
	TotalFailed    int64 `json:"total_failed"`
}
