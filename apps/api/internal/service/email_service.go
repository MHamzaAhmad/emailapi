package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	emailapi "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/ses"
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
	ses         ses.Client
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	riverClient *river.Client[pgx.Tx],
	pgRepo *pgrepo.EmailRepository,
	chRepo *chrepo.EmailRepository,
	validator *validation.EmailValidator,
	sesClient ses.Client,
) *EmailService {
	return &EmailService{
		riverClient: riverClient,
		pgRepo:      pgRepo,
		chRepo:      chRepo,
		validator:   validator,
		ses:         sesClient,
	}
}

// SendEmail handles sending an email request.
// Behavior is controlled by the async flag:
// - async=false: Send synchronously, return message_id immediately (user handles retries)
// - async=true: Queue for async processing (we handle retries)
// - With attachments: Always async (must process/scan first)
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

	// Create email record
	email := &domain.Email{
		ID:        emailID,
		UserID:    userID,
		From:      req.From,
		To:        req.To,
		Cc:        req.Cc,
		Bcc:       req.Bcc,
		Subject:   req.Subject,
		Body:      req.Body,
		HTML:      req.Html,
		InReplyTo: req.InReplyTo,
		Status:    domain.EmailStatusPending,
		Metadata:  metadata,
	}

	if req.ScheduledAt != nil {
		t := req.ScheduledAt.AsTime()
		email.ScheduledAt = &t
	}

	// Attachments force async
	if len(req.Attachments) > 0 {
		email.Status = domain.EmailStatusProcessingAttachments
		if err := s.pgRepo.Create(ctx, email); err != nil {
			return nil, fmt.Errorf("failed to create email: %w", err)
		}
		return s.queueWithAttachments(ctx, emailID, req.Attachments)
	}

	// User wants async
	if req.Async {
		email.Status = domain.EmailStatusQueued
		if err := s.pgRepo.Create(ctx, email); err != nil {
			return nil, fmt.Errorf("failed to create email: %w", err)
		}
		return s.queueForSend(ctx, emailID)
	}

	// Sync path: Try to send immediately with timeout
	if err := s.pgRepo.Create(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to create email: %w", err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	messageID, err := s.sendToSES(sendCtx, email)
	if err != nil {
		// Sync failed - return error (user handles retry)
		s.pgRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, err.Error())
		return nil, fmt.Errorf("send failed: %w", err)
	}

	// Success - update and return message_id
	s.pgRepo.UpdateSent(ctx, emailID, messageID, messageID)

	return &emailapi.SendEmailResponse{
		Id:            emailID,
		MessageId:     messageID,
		Status:        emailapi.EmailStatus_EMAIL_STATUS_SENT,
		StatusMessage: "Email sent successfully",
	}, nil
}

// queueWithAttachments queues an email for attachment processing.
func (s *EmailService) queueWithAttachments(ctx context.Context, emailID string, attachments []*emailapi.Attachment) (*emailapi.SendEmailResponse, error) {
	var workerAttachments []worker.AttachmentSource
	for _, att := range attachments {
		source := worker.AttachmentSource{
			Filename:    att.Filename,
			ContentType: att.ContentType,
		}
		switch src := att.Source.(type) {
		case *emailapi.Attachment_Url:
			source.URL = src.Url
		case *emailapi.Attachment_Base64Content:
			source.Base64Content = src.Base64Content
		}
		workerAttachments = append(workerAttachments, source)
	}

	_, err := s.riverClient.Insert(ctx, worker.ProcessAttachmentsArgs{
		EmailID:     emailID,
		Attachments: workerAttachments,
	}, nil)
	if err != nil {
		s.pgRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, fmt.Sprintf("Failed to enqueue: %v", err))
		return nil, fmt.Errorf("failed to enqueue attachment job: %w", err)
	}

	return &emailapi.SendEmailResponse{
		Id:            emailID,
		Status:        emailapi.EmailStatus_EMAIL_STATUS_PROCESSING_ATTACHMENTS,
		StatusMessage: "Processing attachments before sending",
	}, nil
}

// queueForSend queues an email for async sending.
func (s *EmailService) queueForSend(ctx context.Context, emailID string) (*emailapi.SendEmailResponse, error) {
	_, err := s.riverClient.Insert(ctx, worker.SendEmailArgs{
		EmailID: emailID,
	}, nil)
	if err != nil {
		s.pgRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, fmt.Sprintf("Failed to enqueue: %v", err))
		return nil, fmt.Errorf("failed to enqueue send job: %w", err)
	}

	return &emailapi.SendEmailResponse{
		Id:            emailID,
		Status:        emailapi.EmailStatus_EMAIL_STATUS_QUEUED,
		StatusMessage: "Email queued for sending",
	}, nil
}

// sendToSES sends the email to SES and returns the message ID.
func (s *EmailService) sendToSES(ctx context.Context, email *domain.Email) (string, error) {
	// Build destination
	dest := &types.Destination{
		ToAddresses:  email.To,
		CcAddresses:  email.Cc,
		BccAddresses: email.Bcc,
	}

	// Build content
	var body types.Body
	if email.HTML != "" {
		body.Html = &types.Content{
			Data:    aws.String(email.HTML),
			Charset: aws.String("UTF-8"),
		}
	}
	if email.Body != "" {
		body.Text = &types.Content{
			Data:    aws.String(email.Body),
			Charset: aws.String("UTF-8"),
		}
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(email.From),
		Destination:      dest,
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(email.Subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &body,
			},
		},
	}

	// Add reply-to headers if present
	if email.InReplyTo != "" {
		input.EmailTags = append(input.EmailTags, types.MessageTag{
			Name:  aws.String("InReplyTo"),
			Value: aws.String(email.InReplyTo),
		})
	}

	output, err := s.ses.SendEmail(ctx, input)
	if err != nil {
		return "", err
	}

	return aws.ToString(output.MessageId), nil
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
		MessageId:    email.MessageID,
		InReplyTo:    email.InReplyTo,
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
