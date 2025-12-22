package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	emailapi "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/external/ses"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	"github.com/emailapi/api/internal/validation"
	"github.com/emailapi/api/internal/worker"
)

// EmailService handles email sending operations.
// This is a stateless, compliance-first service - no email content is stored permanently.
type EmailService struct {
	emailapi.UnimplementedEmailServiceServer
	riverClient *river.Client[pgx.Tx]
	chRepo      *chrepo.EmailRepository
	validator   *validation.EmailValidator
	ses         ses.Client
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	riverClient *river.Client[pgx.Tx],
	chRepo *chrepo.EmailRepository,
	validator *validation.EmailValidator,
	sesClient ses.Client,
) *EmailService {
	return &EmailService{
		riverClient: riverClient,
		chRepo:      chRepo,
		validator:   validator,
		ses:         sesClient,
	}
}

// SendEmail handles sending an email request.
// Behavior is controlled by the async flag:
// - async=false + no attachments: Send synchronously, return message_id immediately
// - async=true OR has attachments: Queue for async processing with retries
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
	metadata := make(map[string]string)
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	// Handle attachments - always async
	if len(req.Attachments) > 0 {
		return s.queueWithAttachments(ctx, emailID, userID, req)
	}

	// Handle async flag
	if req.Async {
		return s.queueForSend(ctx, emailID, userID, req)
	}

	// Sync path: Send immediately with timeout
	sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	messageID, err := s.sendToSES(sendCtx, req)
	if err != nil {
		// Log failure
		s.logActivity(ctx, userID, emailID, "failed", err.Error(), req)
		return nil, fmt.Errorf("send failed: %w", err)
	}

	// Write routing entry for reply tracking
	if s.chRepo != nil {
		_ = s.chRepo.InsertRouting(ctx, messageID, emailID, userID)
	}

	// Log success
	s.logActivity(ctx, userID, emailID, "sent", fmt.Sprintf("Message ID: %s", messageID), req)

	return &emailapi.SendEmailResponse{
		Id:            emailID,
		MessageId:     messageID,
		Status:        emailapi.EmailStatus_EMAIL_STATUS_SENT,
		StatusMessage: "Email sent successfully",
	}, nil
}

// queueWithAttachments queues an email with attachments for processing.
func (s *EmailService) queueWithAttachments(ctx context.Context, emailID, userID string, req *emailapi.SendEmailRequest) (*emailapi.SendEmailResponse, error) {
	var attachments []worker.AttachmentSource
	for _, att := range req.Attachments {
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
		attachments = append(attachments, source)
	}

	args := worker.ProcessAttachmentsArgs{
		EmailID:     emailID,
		UserID:      userID,
		From:        req.From,
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     req.Subject,
		Body:        req.Body,
		HTML:        req.Html,
		InReplyTo:   req.InReplyTo,
		References:  req.References,
		Metadata:    req.Metadata,
		Attachments: attachments,
	}

	_, err := s.riverClient.Insert(ctx, args, nil)
	if err != nil {
		s.logActivity(ctx, userID, emailID, "failed", fmt.Sprintf("Failed to enqueue: %v", err), req)
		return nil, fmt.Errorf("failed to enqueue attachment job: %w", err)
	}

	return &emailapi.SendEmailResponse{
		Id:            emailID,
		Status:        emailapi.EmailStatus_EMAIL_STATUS_QUEUED,
		StatusMessage: "Processing attachments",
	}, nil
}

// queueForSend queues an email for async sending.
func (s *EmailService) queueForSend(ctx context.Context, emailID, userID string, req *emailapi.SendEmailRequest) (*emailapi.SendEmailResponse, error) {
	args := worker.SendEmailArgs{
		EmailID:    emailID,
		UserID:     userID,
		From:       req.From,
		To:         req.To,
		Cc:         req.Cc,
		Bcc:        req.Bcc,
		Subject:    req.Subject,
		Body:       req.Body,
		HTML:       req.Html,
		InReplyTo:  req.InReplyTo,
		References: req.References,
		Metadata:   req.Metadata,
	}

	_, err := s.riverClient.Insert(ctx, args, nil)
	if err != nil {
		s.logActivity(ctx, userID, emailID, "failed", fmt.Sprintf("Failed to enqueue: %v", err), req)
		return nil, fmt.Errorf("failed to enqueue send job: %w", err)
	}

	return &emailapi.SendEmailResponse{
		Id:            emailID,
		Status:        emailapi.EmailStatus_EMAIL_STATUS_QUEUED,
		StatusMessage: "Email queued for sending",
	}, nil
}

// sendToSES sends the email to SES and returns the message ID.
func (s *EmailService) sendToSES(ctx context.Context, req *emailapi.SendEmailRequest) (string, error) {
	dest := &types.Destination{
		ToAddresses:  req.To,
		CcAddresses:  req.Cc,
		BccAddresses: req.Bcc,
	}

	var body types.Body
	if req.Html != "" {
		body.Html = &types.Content{
			Data:    aws.String(req.Html),
			Charset: aws.String("UTF-8"),
		}
	}
	if req.Body != "" {
		body.Text = &types.Content{
			Data:    aws.String(req.Body),
			Charset: aws.String("UTF-8"),
		}
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(req.From),
		Destination:      dest,
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(req.Subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &body,
			},
		},
	}

	// Add reply-to headers if present
	if req.InReplyTo != "" {
		input.EmailTags = append(input.EmailTags, types.MessageTag{
			Name:  aws.String("InReplyTo"),
			Value: aws.String(req.InReplyTo),
		})
	}

	if len(req.References) > 0 {
		input.EmailTags = append(input.EmailTags, types.MessageTag{
			Name:  aws.String("References"),
			Value: aws.String(strings.Join(req.References, " ")),
		})
	}

	output, err := s.ses.SendEmail(ctx, input)
	if err != nil {
		return "", err
	}

	return aws.ToString(output.MessageId), nil
}

// logActivity logs an activity for debugging (compliant - no email content stored).
func (s *EmailService) logActivity(ctx context.Context, userID, emailID, action, details string, req *emailapi.SendEmailRequest) {
	if s.chRepo == nil {
		return
	}

	status := "success"
	if action == "failed" {
		status = "failed"
	}

	metadata := map[string]interface{}{
		"from":            req.From,
		"to":              req.To,
		"subject":         req.Subject,
		"has_attachments": len(req.Attachments) > 0,
		"async":           req.Async,
	}

	s.chRepo.LogEmailEvent(ctx, userID, emailID, action, status, details, metadata)
}
