package service

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	emailapi "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/eventstream"
	"github.com/emailapi/api/internal/external/ses"
	"github.com/emailapi/api/internal/validation"
	"github.com/emailapi/api/internal/webhook"
	"github.com/emailapi/api/internal/worker"
)

// EmailService handles email sending operations.
// This is a stateless, compliance-first service - no email content is stored permanently.
type EmailService struct {
	queue             QueueClient
	analytics         Analytics
	validator         SenderValidator
	ses               ses.Client
	webhookSender     webhook.Sender
	eventConsumer     eventstream.Consumer
	reputationChecker validation.ReputationChecker
	unsubscribeSvc    UnsubscribeManager
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	queue QueueClient,
	analytics Analytics,
	validator SenderValidator,
	sesClient ses.Client,
	webhookSender webhook.Sender,
	eventConsumer eventstream.Consumer,
	reputationChecker validation.ReputationChecker,
	unsubscribeSvc UnsubscribeManager,
) *EmailService {
	return &EmailService{
		queue:             queue,
		analytics:         analytics,
		validator:         validator,
		ses:               sesClient,
		webhookSender:     webhookSender,
		eventConsumer:     eventConsumer,
		reputationChecker: reputationChecker,
		unsubscribeSvc:    unsubscribeSvc,
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

	// Check for dry-run mode (set by handler from X-Dry-Run header)
	dryRun, _ := ctx.Value("dry_run").(bool)

	// Validate all email addresses and body content in parallel
	if s.validator != nil {
		if err := s.validator.ValidateSendEmail(ctx, userID, req.From, req.To, req.Cc, req.Bcc, req.Body, req.Html); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	emailID := uuid.New().String()

	// Check unsubscribe list and filter out unsubscribed recipients
	if s.unsubscribeSvc != nil {
		allRecipients := append(append([]string{}, req.To...), req.Cc...)
		allRecipients = append(allRecipients, req.Bcc...)

		unsubscribed, err := s.unsubscribeSvc.CheckBatch(ctx, userID, allRecipients)
		if err != nil {
			// Log but don't fail - unsubscribe check is not critical path
			fmt.Printf("Warning: failed to check unsubscribes: %v\n", err)
		} else if len(unsubscribed) > 0 {
			// Filter out unsubscribed recipients
			unsubSet := make(map[string]bool, len(unsubscribed))
			for _, email := range unsubscribed {
				unsubSet[strings.ToLower(strings.TrimSpace(email))] = true
			}

			req.To = filterEmails(req.To, unsubSet)
			req.Cc = filterEmails(req.Cc, unsubSet)
			req.Bcc = filterEmails(req.Bcc, unsubSet)

			// If no recipients left, return early
			if len(req.To) == 0 && len(req.Cc) == 0 && len(req.Bcc) == 0 {
				return &emailapi.SendEmailResponse{
					Id:            emailID,
					Status:        emailapi.EmailStatus_EMAIL_STATUS_SENT,
					StatusMessage: "All recipients have unsubscribed",
				}, nil
			}
		}

		// Replace {{unsubscribe_link}} placeholder for SYNC single-recipient emails only
		// Async emails are handled by the worker which splits multi-recipient into individual sends
		hasPlaceholder := strings.Contains(req.Body, "{{unsubscribe_link}}") || strings.Contains(req.Html, "{{unsubscribe_link}}")
		totalRecipients := len(req.To) + len(req.Cc) + len(req.Bcc)
		isAsync := req.Async || len(req.Attachments) > 0 || req.ScheduledAt != nil

		if hasPlaceholder && !isAsync && totalRecipients == 1 && len(req.To) == 1 {
			// Sync + single recipient - generate unique link for them
			unsubLink, err := s.unsubscribeSvc.GenerateLink(userID, req.To[0], emailID)
			if err == nil {
				req.Body = strings.ReplaceAll(req.Body, "{{unsubscribe_link}}", unsubLink)
				req.Html = strings.ReplaceAll(req.Html, "{{unsubscribe_link}}", unsubLink)
			}
		} else if hasPlaceholder && !isAsync && totalRecipients > 1 {
			// Sync + multi-recipient = skip placeholder, warn user to use async
			fmt.Printf("Warning: {{unsubscribe_link}} with %d recipients in sync mode - use async for automatic splitting\n", totalRecipients)
		}
		// For async/scheduled/attachments with placeholders: worker handles splitting
	}

	// Convert metadata
	metadata := make(map[string]string)
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	// Handle scheduled emails - queue with delay
	if req.ScheduledAt != nil {
		return s.queueScheduled(ctx, emailID, userID, req, dryRun)
	}

	// Handle attachments - always async
	if len(req.Attachments) > 0 {
		return s.queueWithAttachments(ctx, emailID, userID, req, dryRun)
	}

	// Handle async flag
	if req.Async {
		return s.queueForSend(ctx, emailID, userID, req, dryRun)
	}

	// Sync path: Handle dry-run or send immediately
	if dryRun {
		// Simulate SES latency (50-150ms)
		delay := 50*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
		time.Sleep(delay)

		fakeMessageID := fmt.Sprintf("dry-run-%s@simpleemailapi.dev", emailID[:8])
		return &emailapi.SendEmailResponse{
			Id:            emailID,
			MessageId:     fakeMessageID,
			Status:        emailapi.EmailStatus_EMAIL_STATUS_SENT,
			StatusMessage: "[DRY-RUN] Email simulated successfully",
		}, nil
	}

	// Send immediately with timeout
	sendCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	messageID, err := s.sendToSES(sendCtx, req)
	if err != nil {
		// Log failure
		s.logActivity(ctx, userID, emailID, "failed", err.Error(), req)
		return nil, fmt.Errorf("send failed: %w", err)
	}

	// Write routing entry for reply tracking (async - don't block response)
	if s.analytics != nil {
		go func() {
			if err := s.analytics.Email().InsertRouting(context.Background(), messageID, emailID, userID); err != nil {
				// Log but don't fail - routing is for reply tracking, not critical path
				fmt.Printf("Warning: failed to insert routing entry: %v\n", err)
			}
		}()
	}

	// Log success (async - don't block response)
	go s.logActivity(context.Background(), userID, emailID, "sent", fmt.Sprintf("Message ID: %s", messageID), req)

	// Send webhook notification (already async)
	s.sendWebhook(ctx, userID, emailID, messageID, req, "sent")

	return &emailapi.SendEmailResponse{
		Id:            emailID,
		MessageId:     messageID,
		Status:        emailapi.EmailStatus_EMAIL_STATUS_SENT,
		StatusMessage: "Email sent successfully",
	}, nil
}

// queueWithAttachments queues an email with attachments for processing.
func (s *EmailService) queueWithAttachments(ctx context.Context, emailID, userID string, req *emailapi.SendEmailRequest, dryRun bool) (*emailapi.SendEmailResponse, error) {
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
		DryRun:      dryRun,
	}

	// Pass unsubscribe config if placeholder is present (for worker to split multi-recipient emails)
	if s.unsubscribeSvc != nil && (strings.Contains(req.Body, "{{unsubscribe_link}}") || strings.Contains(req.Html, "{{unsubscribe_link}}")) {
		args.UnsubscribeBaseURL = s.unsubscribeSvc.BaseURL()
		args.UnsubscribeTokenSecret = s.unsubscribeSvc.TokenSecret()
	}

	// Add scheduled time if present
	if req.ScheduledAt != nil {
		scheduledTime := req.ScheduledAt.AsTime()
		args.ScheduledAt = &scheduledTime
	}

	_, err := s.queue.Insert(ctx, args, nil)
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
func (s *EmailService) queueForSend(ctx context.Context, emailID, userID string, req *emailapi.SendEmailRequest, dryRun bool) (*emailapi.SendEmailResponse, error) {
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
		DryRun:     dryRun,
	}

	// Pass unsubscribe config if placeholder is present (for worker to split multi-recipient emails)
	if s.unsubscribeSvc != nil && (strings.Contains(req.Body, "{{unsubscribe_link}}") || strings.Contains(req.Html, "{{unsubscribe_link}}")) {
		args.UnsubscribeBaseURL = s.unsubscribeSvc.BaseURL()
		args.UnsubscribeTokenSecret = s.unsubscribeSvc.TokenSecret()
	}

	_, err := s.queue.Insert(ctx, args, nil)
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

// queueScheduled queues an email for scheduled delivery.
func (s *EmailService) queueScheduled(ctx context.Context, emailID, userID string, req *emailapi.SendEmailRequest, dryRun bool) (*emailapi.SendEmailResponse, error) {
	// Convert protobuf timestamp to Go time
	scheduledTime := req.ScheduledAt.AsTime()

	// Validate scheduled time is in the future
	if scheduledTime.Before(time.Now()) {
		return nil, fmt.Errorf("scheduled_at must be in the future")
	}

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
		DryRun:     dryRun,
	}

	// Pass unsubscribe config if placeholder is present (for worker to split multi-recipient emails)
	if s.unsubscribeSvc != nil && (strings.Contains(req.Body, "{{unsubscribe_link}}") || strings.Contains(req.Html, "{{unsubscribe_link}}")) {
		args.UnsubscribeBaseURL = s.unsubscribeSvc.BaseURL()
		args.UnsubscribeTokenSecret = s.unsubscribeSvc.TokenSecret()
	}

	// Queue with scheduled time
	_, err := s.queue.Insert(ctx, args, &river.InsertOpts{
		ScheduledAt: scheduledTime,
	})
	if err != nil {
		s.logActivity(ctx, userID, emailID, "failed", fmt.Sprintf("Failed to schedule: %v", err), req)
		return nil, fmt.Errorf("failed to schedule email: %w", err)
	}

	return &emailapi.SendEmailResponse{
		Id:            emailID,
		Status:        emailapi.EmailStatus_EMAIL_STATUS_QUEUED,
		StatusMessage: fmt.Sprintf("Email scheduled for %s", scheduledTime.Format(time.RFC3339)),
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
	if s.analytics == nil {
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

	s.analytics.Email().LogEmailEvent(ctx, userID, emailID, action, status, details, metadata)
}

// sendWebhook sends a webhook notification for email events.
func (s *EmailService) sendWebhook(ctx context.Context, userID, emailID, messageID string, req *emailapi.SendEmailRequest, status string) {
	if s.webhookSender == nil {
		return
	}

	event := &emailapi.EmailSentEvent{
		EmailId:   emailID,
		UserId:    userID,
		MessageId: messageID,
		From:      req.From,
		To:        req.To,
		Cc:        req.Cc,
		Bcc:       req.Bcc,
		Subject:   req.Subject,
		Metadata:  req.Metadata,
	}

	s.webhookSender.SendEmailSent(ctx, userID, event)
}

// StreamEvents returns a channel of events for the user using Redis Consumer Groups.
// The apiKeyID is used as the consumer identifier for tracking acknowledgment state.
// Unacknowledged events are automatically replayed on reconnect.
func (s *EmailService) StreamEvents(ctx context.Context, userID, apiKeyID string, eventTypes []emailapi.EventType, batchSize int32) (<-chan *emailapi.Event, error) {
	// Check reputation - suspended users cannot access streaming API
	if s.reputationChecker != nil {
		if err := s.reputationChecker.CheckSendPermission(ctx, userID); err != nil {
			return nil, err
		}
	}

	if s.eventConsumer == nil {
		return nil, fmt.Errorf("event streaming not configured")
	}
	return s.eventConsumer.Subscribe(ctx, userID, apiKeyID, eventTypes, batchSize)
}

// AckEvents acknowledges events as processed, preventing replay on reconnect.
func (s *EmailService) AckEvents(ctx context.Context, userID string, eventIDs []string) (int64, error) {
	// Check reputation - suspended users cannot ack events
	if s.reputationChecker != nil {
		if err := s.reputationChecker.CheckSendPermission(ctx, userID); err != nil {
			return 0, err
		}
	}

	if s.eventConsumer == nil {
		return 0, fmt.Errorf("event streaming not configured")
	}
	return s.eventConsumer.Ack(ctx, userID, eventIDs)
}

// filterEmails removes emails that are in the exclusion set.
func filterEmails(emails []string, exclude map[string]bool) []string {
	if len(emails) == 0 || len(exclude) == 0 {
		return emails
	}
	result := make([]string, 0, len(emails))
	for _, email := range emails {
		if !exclude[strings.ToLower(strings.TrimSpace(email))] {
			result = append(result, email)
		}
	}
	return result
}
