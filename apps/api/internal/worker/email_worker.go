package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
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

	// 7. Log sent event to ClickHouse
	if err := w.chRepo.AddEmailEvent(ctx, email, "sent"); err != nil {
		fmt.Printf("Warning: failed to log sent event: %v\n", err)
	}

	// 8. Delete from PostgreSQL (cleanup)
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

// sendEmail sends the email via SES with attachments.
func (w *EmailWorker) sendEmail(ctx context.Context, email *domain.Email, attachments []attachmentContent) (string, error) {
	// Build MIME message if we have attachments
	var rawMessage []byte
	var err error

	if len(attachments) > 0 {
		rawMessage, err = buildMIMEMessage(email, attachments)
		if err != nil {
			return "", fmt.Errorf("failed to build MIME message: %w", err)
		}
	}

	// Build recipient list
	var toAddresses []string
	toAddresses = append(toAddresses, email.To...)
	if len(email.Cc) > 0 {
		toAddresses = append(toAddresses, email.Cc...)
	}
	if len(email.Bcc) > 0 {
		toAddresses = append(toAddresses, email.Bcc...)
	}

	var input *sesv2.SendEmailInput

	if len(attachments) > 0 {
		// Send raw MIME message for attachments
		input = &sesv2.SendEmailInput{
			Content: &types.EmailContent{
				Raw: &types.RawMessage{
					Data: rawMessage,
				},
			},
		}
	} else {
		// Simple email without attachments
		input = &sesv2.SendEmailInput{
			FromEmailAddress: aws.String(email.From),
			Destination: &types.Destination{
				ToAddresses:  email.To,
				CcAddresses:  email.Cc,
				BccAddresses: email.Bcc,
			},
			Content: &types.EmailContent{
				Simple: &types.Message{
					Subject: &types.Content{
						Data: aws.String(email.Subject),
					},
					Body: &types.Body{},
				},
			},
		}

		if email.HTML != "" {
			input.Content.Simple.Body.Html = &types.Content{
				Data: aws.String(email.HTML),
			}
		}
		if email.Body != "" {
			input.Content.Simple.Body.Text = &types.Content{
				Data: aws.String(email.Body),
			}
		}
	}

	resp, err := w.sesClient.SendEmail(ctx, input)
	if err != nil {
		return "", err
	}

	return aws.ToString(resp.MessageId), nil
}

// buildMIMEMessage constructs a MIME message with attachments.
func buildMIMEMessage(email *domain.Email, attachments []attachmentContent) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Write headers
	fmt.Fprintf(&buf, "From: %s\r\n", email.From)
	fmt.Fprintf(&buf, "To: %s\r\n", joinAddresses(email.To))
	if len(email.Cc) > 0 {
		fmt.Fprintf(&buf, "Cc: %s\r\n", joinAddresses(email.Cc))
	}
	fmt.Fprintf(&buf, "Subject: %s\r\n", email.Subject)
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", writer.Boundary())

	// Write text/html body
	if email.HTML != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/html; charset=UTF-8")
		part, _ := writer.CreatePart(h)
		part.Write([]byte(email.HTML))
	} else if email.Body != "" {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "text/plain; charset=UTF-8")
		part, _ := writer.CreatePart(h)
		part.Write([]byte(email.Body))
	}

	// Write attachments
	for _, att := range attachments {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", att.ContentType)
		h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))
		h.Set("Content-Transfer-Encoding", "base64")
		part, _ := writer.CreatePart(h)
		// Write base64 encoded content
		encoded := base64.StdEncoding.EncodeToString(att.Data)
		part.Write([]byte(encoded))
	}

	writer.Close()
	return buf.Bytes(), nil
}

func joinAddresses(addrs []string) string {
	if len(addrs) == 0 {
		return ""
	}
	result := addrs[0]
	for i := 1; i < len(addrs); i++ {
		result += ", " + addrs[i]
	}
	return result
}
