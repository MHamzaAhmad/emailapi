package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	emailapi "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	pgrepo "github.com/emailapi/api/internal/repository/postgres"
	"github.com/emailapi/api/internal/validation"
	"github.com/emailapi/api/internal/worker"
)

// EmailService handles email-related operations.
type EmailService struct {
	emailapi.UnimplementedEmailServiceServer
	riverClient *river.Client[pgx.Tx]
	pgRepo      *pgrepo.EmailRepository
	chRepo      *chrepo.EmailRepository
	validator   *validation.EmailValidator
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	riverClient *river.Client[pgx.Tx],
	pgRepo *pgrepo.EmailRepository,
	chRepo *chrepo.EmailRepository,
	validator *validation.EmailValidator,
) *EmailService {
	return &EmailService{
		riverClient: riverClient,
		pgRepo:      pgRepo,
		chRepo:      chRepo,
		validator:   validator,
	}
}

// SendEmail handles sending an email request.
// 1. Validates all email addresses (FROM domain ownership, TO/CC/BCC validity)
// 2. Saves email to PostgreSQL with pending status
// 3. Enqueues attachment processing job (if attachments) or send job (if no attachments)
// 4. Returns immediately with email ID
func (s *EmailService) SendEmail(ctx context.Context, req *emailapi.SendEmailRequest) (*emailapi.SendEmailResponse, error) {
	// Get user ID from context (set by auth interceptor)
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("user_id not found in context")
	}

	// Validate all email addresses before processing
	if s.validator != nil {
		if err := s.validator.ValidateSendEmail(ctx, userID, req.From, req.To, req.Cc, req.Bcc); err != nil {
			return nil, fmt.Errorf("email validation failed: %w", err)
		}
	}

	emailID := uuid.New().String()

	// Convert metadata
	metadata := make(domain.Metadata)
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	// Determine initial status
	status := domain.EmailStatusPending
	if len(req.Attachments) > 0 {
		status = domain.EmailStatusProcessingAttachments
	}

	// Create email record in PostgreSQL
	email := &domain.Email{
		ID:       emailID,
		UserID:   userID,
		From:     req.From,
		To:       req.To,
		Cc:       req.Cc,
		Bcc:      req.Bcc,
		Subject:  req.Subject,
		Body:     req.Body,
		HTML:     req.Html,
		Status:   status,
		Metadata: metadata,
	}

	if req.ScheduledAt != nil {
		t := req.ScheduledAt.AsTime()
		email.ScheduledAt = &t
	}

	if err := s.pgRepo.Create(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to create email: %w", err)
	}

	// Enqueue appropriate job
	if len(req.Attachments) > 0 {
		// Convert attachments to worker format
		var attachments []worker.AttachmentSource
		for _, att := range req.Attachments {
			source := worker.AttachmentSource{
				Filename:    att.Filename,
				ContentType: att.ContentType,
			}
			// Handle oneof source
			switch src := att.Source.(type) {
			case *emailapi.Attachment_Url:
				source.URL = src.Url
			case *emailapi.Attachment_Base64Content:
				source.Base64Content = src.Base64Content
			}
			attachments = append(attachments, source)
		}

		// Enqueue attachment processing job
		_, err := s.riverClient.Insert(ctx, worker.ProcessAttachmentsArgs{
			EmailID:     emailID,
			Attachments: attachments,
		}, nil)
		if err != nil {
			// Update status to failed
			s.pgRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, fmt.Sprintf("Failed to enqueue attachment job: %v", err))
			return nil, fmt.Errorf("failed to enqueue attachment job: %w", err)
		}
	} else {
		// No attachments - enqueue send job directly
		_, err := s.riverClient.Insert(ctx, worker.SendEmailArgs{
			EmailID: emailID,
		}, nil)
		if err != nil {
			s.pgRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, fmt.Sprintf("Failed to enqueue send job: %v", err))
			return nil, fmt.Errorf("failed to enqueue send job: %w", err)
		}

		// Update status to queued
		s.pgRepo.UpdateStatus(ctx, emailID, domain.EmailStatusQueued, "")
	}

	return &emailapi.SendEmailResponse{
		Id:     emailID,
		Status: emailapi.EmailStatus_EMAIL_STATUS_PENDING,
	}, nil
}

// GetEmail retrieves an email by ID using cascading lookup (PG → CH → 404).
func (s *EmailService) GetEmail(ctx context.Context, req *emailapi.GetEmailRequest) (*emailapi.Email, error) {
	// 1. Try PostgreSQL first (active emails, faster for point lookups)
	email, err := s.pgRepo.GetByID(ctx, req.Id)
	if err == nil {
		// Get attachments
		attachments, _ := s.pgRepo.GetAttachmentsByEmailID(ctx, req.Id)
		return domainEmailToProto(email, attachments), nil
	}

	// 2. Fallback to ClickHouse (archived emails)
	archivedEmail, err := s.chRepo.GetArchivedEmail(ctx, req.Id)
	if err == nil {
		return domainEmailToProto(archivedEmail, nil), nil
	}

	// 3. Not found in either store
	return nil, errors.New("email not found")
}

// ListEmails retrieves emails by category: ACTIVE (PostgreSQL) or ARCHIVED (ClickHouse).
func (s *EmailService) ListEmails(ctx context.Context, req *emailapi.ListEmailsRequest) (*emailapi.ListEmailsResponse, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("user_id not found in context")
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := int(req.Offset)

	// Default to ACTIVE if not specified
	category := req.Category
	if category == emailapi.EmailCategory_EMAIL_CATEGORY_UNSPECIFIED {
		category = emailapi.EmailCategory_EMAIL_CATEGORY_ACTIVE
	}

	var protoEmails []*emailapi.Email
	var totalCount int
	var err error

	switch category {
	case emailapi.EmailCategory_EMAIL_CATEGORY_ACTIVE:
		// Query PostgreSQL for active emails
		emails, qErr := s.pgRepo.GetByUserID(ctx, userID, limit, offset)
		if qErr != nil {
			return nil, fmt.Errorf("failed to get active emails: %w", qErr)
		}
		for _, email := range emails {
			protoEmails = append(protoEmails, domainEmailToProto(email, nil))
		}
		totalCount, err = s.pgRepo.CountByUserID(ctx, userID)

	case emailapi.EmailCategory_EMAIL_CATEGORY_ARCHIVED:
		// Query ClickHouse for archived emails
		emails, qErr := s.chRepo.ListArchivedEmails(ctx, userID, limit, offset)
		if qErr != nil {
			return nil, fmt.Errorf("failed to get archived emails: %w", qErr)
		}
		for _, email := range emails {
			protoEmails = append(protoEmails, domainEmailToProto(email, nil))
		}
		totalCount, err = s.chRepo.CountArchivedEmails(ctx, userID)

	default:
		return nil, fmt.Errorf("invalid category: %v", category)
	}

	if err != nil {
		// Non-fatal, just log and continue
		totalCount = len(protoEmails)
	}

	return &emailapi.ListEmailsResponse{
		Data:       protoEmails,
		Limit:      int32(limit),
		Offset:     int32(offset),
		TotalCount: int32(totalCount),
		Category:   category,
	}, nil
}

// domainEmailToProto converts a domain Email to proto Email.
func domainEmailToProto(email *domain.Email, attachments []*domain.EmailAttachment) *emailapi.Email {
	protoEmail := &emailapi.Email{
		Id:           email.ID,
		From:         email.From,
		To:           email.To,
		Cc:           email.Cc,
		Bcc:          email.Bcc,
		Subject:      email.Subject,
		Body:         email.Body,
		Html:         email.HTML,
		Status:       domainStatusToProto(email.Status),
		ProviderId:   email.ProviderID,
		UserId:       email.UserID,
		ErrorMessage: email.ErrorMessage,
	}

	// Convert metadata
	if email.Metadata != nil {
		protoEmail.Metadata = make(map[string]string)
		for k, v := range email.Metadata {
			if str, ok := v.(string); ok {
				protoEmail.Metadata[k] = str
			} else {
				// Convert non-string values to JSON
				b, _ := json.Marshal(v)
				protoEmail.Metadata[k] = string(b)
			}
		}
	}

	// Convert attachments
	for _, att := range attachments {
		protoEmail.Attachments = append(protoEmail.Attachments, &emailapi.EmailAttachment{
			Id:          att.ID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			SizeBytes:   att.SizeBytes,
			ScanStatus:  string(att.ScanStatus),
		})
	}

	return protoEmail
}

// domainStatusToProto converts domain EmailStatus to proto EmailStatus.
func domainStatusToProto(status domain.EmailStatus) emailapi.EmailStatus {
	switch status {
	case domain.EmailStatusPending:
		return emailapi.EmailStatus_EMAIL_STATUS_PENDING
	case domain.EmailStatusProcessingAttachments:
		return emailapi.EmailStatus_EMAIL_STATUS_PROCESSING_ATTACHMENTS
	case domain.EmailStatusScanningAttachments:
		return emailapi.EmailStatus_EMAIL_STATUS_SCANNING_ATTACHMENTS
	case domain.EmailStatusScanFailed:
		return emailapi.EmailStatus_EMAIL_STATUS_SCAN_FAILED
	case domain.EmailStatusQueued:
		return emailapi.EmailStatus_EMAIL_STATUS_QUEUED
	case domain.EmailStatusSent:
		return emailapi.EmailStatus_EMAIL_STATUS_SENT
	case domain.EmailStatusDelivered:
		return emailapi.EmailStatus_EMAIL_STATUS_DELIVERED
	case domain.EmailStatusFailed:
		return emailapi.EmailStatus_EMAIL_STATUS_FAILED
	case domain.EmailStatusBounced:
		return emailapi.EmailStatus_EMAIL_STATUS_BOUNCED
	default:
		return emailapi.EmailStatus_EMAIL_STATUS_UNSPECIFIED
	}
}
