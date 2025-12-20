package domain

import (
	"time"
)

// EmailStatus represents the current state of an email.
type EmailStatus string

const (
	EmailStatusPending   EmailStatus = "pending"
	EmailStatusSent      EmailStatus = "sent"
	EmailStatusDelivered EmailStatus = "delivered"
	EmailStatusFailed    EmailStatus = "failed"
	EmailStatusBounced   EmailStatus = "bounced"
)

// Email represents an email entity in the system.
type Email struct {
	ID          string      `json:"id"`
	MessageID   string      `json:"message_id,omitempty"`
	From        string      `json:"from"`
	To          []string    `json:"to"`
	Cc          []string    `json:"cc,omitempty"`
	Bcc         []string    `json:"bcc,omitempty"`
	Subject     string      `json:"subject"`
	Body        string      `json:"body"`
	HTML        string      `json:"html,omitempty"`
	Status      EmailStatus `json:"status"`
	ProviderID  string      `json:"provider_id,omitempty"`
	UserID      string      `json:"user_id"`
	WebhookID   string      `json:"webhook_id,omitempty"`
	Metadata    Metadata    `json:"metadata,omitempty"`
	ScheduledAt *time.Time  `json:"scheduled_at,omitempty"`
	SentAt      *time.Time  `json:"sent_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Metadata holds additional key-value data for emails.
type Metadata map[string]interface{}

// Attachment represents an email attachment.
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content"`
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
