package worker

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/jordan-wright/email"
	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/external/s3"
	"github.com/emailapi/api/internal/external/ses"
	chrepo "github.com/emailapi/api/internal/repository/clickhouse"
	pgrepo "github.com/emailapi/api/internal/repository/postgres"
)

// SendEmailArgs defines the arguments for the send_email job.
// Now only needs EmailID since email data is stored in PostgreSQL.
type SendEmailArgs struct {
	EmailID string `json:"email_id"`
}

func (SendEmailArgs) Kind() string { return "send_email" }

// EmailWorker handles email sending jobs.
type EmailWorker struct {
	river.WorkerDefaults[SendEmailArgs]
	sesClient ses.Client
	s3Client  s3.Client
	pgRepo    *pgrepo.EmailRepository
	chRepo    *chrepo.EmailRepository
}

// NewEmailWorker creates a new EmailWorker.
func NewEmailWorker(
	sesClient ses.Client,
	s3Client s3.Client,
	pgRepo *pgrepo.EmailRepository,
	chRepo *chrepo.EmailRepository,
) *EmailWorker {
	return &EmailWorker{
		sesClient: sesClient,
		s3Client:  s3Client,
		pgRepo:    pgRepo,
		chRepo:    chRepo,
	}
}

func (w *EmailWorker) Work(ctx context.Context, job *river.Job[SendEmailArgs]) error {
	emailID := job.Args.EmailID

	// 1. Load email from PostgreSQL
	email, err := w.pgRepo.GetByID(ctx, emailID)
	if err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}

	// 2. Load attachments
	attachments, err := w.pgRepo.GetAttachmentsByEmailID(ctx, emailID)
	if err != nil {
		return fmt.Errorf("failed to get attachments: %w", err)
	}
	email.Attachments = make([]domain.EmailAttachment, len(attachments))
	for i, att := range attachments {
		email.Attachments[i] = *att
	}

	// 3. Download attachments from S3
	var attachmentData []attachmentContent
	for _, att := range attachments {
		data, err := w.s3Client.Download(ctx, att.S3Key)
		if err != nil {
			w.pgRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, fmt.Sprintf("Failed to download attachment: %v", err))
			return fmt.Errorf("failed to download attachment %s: %w", att.Filename, err)
		}
		attachmentData = append(attachmentData, attachmentContent{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Data:        data,
		})
	}

	// 4. Send via SES
	messageID, err := w.sendEmail(ctx, email, attachmentData)
	if err != nil {
		w.pgRepo.UpdateStatus(ctx, emailID, domain.EmailStatusFailed, err.Error())
		// Log failure event
		w.chRepo.AddEmailEvent(ctx, email, "failed")
		return fmt.Errorf("failed to send email: %w", err)
	}

	// 5. Update email with message ID and sent status
	email.MessageID = messageID
	email.Status = domain.EmailStatusSent
	now := time.Now()
	email.SentAt = &now

	// 6. Archive to ClickHouse (full email record)
	if err := w.chRepo.ArchiveEmail(ctx, email); err != nil {
		// Log but don't fail the job - email was sent successfully
		fmt.Printf("Warning: failed to archive email to ClickHouse: %v\n", err)
	}

	// 7. Write to routing table for infinite reply tracking
	if err := w.chRepo.InsertRouting(ctx, messageID, emailID, email.UserID, email.From); err != nil {
		fmt.Printf("Warning: failed to insert routing entry: %v\n", err)
	}

	// 8. Log sent event to ClickHouse
	if err := w.chRepo.AddEmailEvent(ctx, email, "sent"); err != nil {
		fmt.Printf("Warning: failed to log sent event: %v\n", err)
	}

	// 9. Delete from PostgreSQL (cleanup)
	if err := w.pgRepo.Delete(ctx, emailID); err != nil {
		// Log but don't fail - email was sent and archived
		fmt.Printf("Warning: failed to delete email from PostgreSQL: %v\n", err)
	}

	fmt.Printf("Sent email %s to %v (MsgID: %s)\n", emailID, email.To, messageID)
	return nil
}

type attachmentContent struct {
	Filename    string
	ContentType string
	Data        []byte
}

// sendEmail sends the email via SES using raw MIME message.
// Uses jordan-wright/email library to build proper MIME with attachments.
func (w *EmailWorker) sendEmail(ctx context.Context, domainEmail *domain.Email, attachments []attachmentContent) (string, error) {
	// Build MIME message using jordan-wright/email
	rawMessage, err := buildMIMEMessage(domainEmail, attachments)
	if err != nil {
		return "", fmt.Errorf("failed to build MIME message: %w", err)
	}

	// Always send as raw email for consistency
	input := &sesv2.SendEmailInput{
		Content: &types.EmailContent{
			Raw: &types.RawMessage{
				Data: rawMessage,
			},
		},
	}

	resp, err := w.sesClient.SendEmail(ctx, input)
	if err != nil {
		return "", err
	}

	return aws.ToString(resp.MessageId), nil
}

// buildMIMEMessage constructs a MIME message using jordan-wright/email library.
func buildMIMEMessage(domainEmail *domain.Email, attachments []attachmentContent) ([]byte, error) {
	e := email.NewEmail()

	// Set sender and recipients
	e.From = domainEmail.From
	e.To = domainEmail.To
	e.Cc = domainEmail.Cc
	e.Bcc = domainEmail.Bcc
	e.Subject = domainEmail.Subject

	// Set body content
	if domainEmail.Body != "" {
		e.Text = []byte(domainEmail.Body)
	}
	if domainEmail.HTML != "" {
		e.HTML = []byte(domainEmail.HTML)
	}

	// Add attachments
	for _, att := range attachments {
		_, err := e.Attach(bytes.NewReader(att.Data), att.Filename, att.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to attach file %s: %w", att.Filename, err)
		}
	}

	// Build the raw MIME message
	return e.Bytes()
}
