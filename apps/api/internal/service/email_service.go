package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/emailapi/api/internal/domain"
)

// EmailService handles email business logic.
type EmailService struct {
	store Store
}

// NewEmailService creates a new EmailService.
func NewEmailService(store Store) *EmailService {
	return &EmailService{store: store}
}

// Send validates and queues an email for sending.
func (s *EmailService) Send(ctx context.Context, userID string, req *domain.SendEmailRequest) (*domain.SendEmailResponse, error) {
	// Validate request
	if err := s.validateSendRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create email entity
	email := &domain.Email{
		ID:          uuid.New().String(),
		From:        req.From,
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     req.Subject,
		Body:        req.Body,
		HTML:        req.HTML,
		Status:      domain.EmailStatusPending,
		UserID:      userID,
		Metadata:    req.Metadata,
		ScheduledAt: req.ScheduledAt,
	}

	// Store email
	if err := s.store.Emails().Create(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to create email: %w", err)
	}

	// TODO: Queue email for async sending via worker
	// For now, we'll just mark it as sent
	email.Status = domain.EmailStatusSent
	email.SentAt = ptr(time.Now())
	if err := s.store.Emails().Update(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to update email status: %w", err)
	}

	return &domain.SendEmailResponse{
		ID:     email.ID,
		Status: email.Status,
	}, nil
}

// GetByID retrieves an email by ID.
func (s *EmailService) GetByID(ctx context.Context, userID, emailID string) (*domain.Email, error) {
	email, err := s.store.Emails().GetByID(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Authorization check
	if email.UserID != userID {
		return nil, fmt.Errorf("email not found")
	}

	return email, nil
}

// List retrieves emails for a user with pagination.
func (s *EmailService) List(ctx context.Context, userID string, limit, offset int) ([]*domain.Email, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.store.Emails().GetByUserID(ctx, userID, limit, offset)
}

// validateSendRequest validates the email send request.
func (s *EmailService) validateSendRequest(req *domain.SendEmailRequest) error {
	if len(req.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if req.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if req.Body == "" && req.HTML == "" {
		return fmt.Errorf("body or html content is required")
	}

	return nil
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}
