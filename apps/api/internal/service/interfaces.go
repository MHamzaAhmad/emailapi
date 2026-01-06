package service

import (
	"context"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

//go:generate mockgen -destination=mocks/mock_service_interfaces.go -package=mocks github.com/emailapi/api/internal/service QueueClient,SenderValidator,UnsubscribeManager,ReputationRecorder,SuppressionManager

// QueueClient defines the interface for queuing background jobs (River).
type QueueClient interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// SenderValidator defines the interface for validating email sending requests.
type SenderValidator interface {
	ValidateSendEmail(ctx context.Context, userID, from string, to, cc, bcc []string, body, html string) error
}

// UnsubscribeManager defines the interface for checking and managing unsubscribes.
type UnsubscribeManager interface {
	CheckBatch(ctx context.Context, userID string, emails []string) ([]string, error)
	GenerateLink(userID, recipientEmail, emailID string) (string, error)
	BaseURL() string
	TokenSecret() string
}

// ReputationRecorder defines the interface for recording reputation incidents.
type ReputationRecorder interface {
	RecordBounceIncident(ctx context.Context, userID, messageID, bounceType, bounceSubType string, recipients []domain.BounceRecipient) error
	RecordComplaintIncident(ctx context.Context, userID, messageID, feedbackType string, recipientEmails []string) error
}

// SuppressionManager defines the interface for suppression management.
type SuppressionManager interface {
	Add(ctx context.Context, entry *suppression.Entry) error
}
