package handlers

//go:generate mockgen -destination=mocks/mock_handlers.go -package=mocks github.com/emailapi/api/internal/events/handlers Analytics,ActivityLogger,EmailRouter,SuppressionManager,ReputationRecorder,InboundEmailProcessor

import (
	"context"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/repository/tinybird"
)

// Analytics interface for activity logging and email routing.
type Analytics interface {
	Activity() ActivityLogger
	Email() EmailRouter
}

// ActivityLogger logs user activities.
type ActivityLogger interface {
	Log(ctx context.Context, userID, entityType, entityID, action, status, message string, metadata map[string]interface{}) error
}

// EmailRouter looks up email routing information.
type EmailRouter interface {
	LookupRouting(ctx context.Context, messageID string) (*tinybird.EmailRouting, error)
}

// SuppressionManager manages email suppression lists.
type SuppressionManager interface {
	Add(ctx context.Context, entry *suppression.Entry) error
}

// ReputationRecorder records reputation incidents.
type ReputationRecorder interface {
	RecordBounceIncident(ctx context.Context, userID, messageID, bounceType, bounceSubType string, recipients []domain.BounceRecipient) error
	RecordComplaintIncident(ctx context.Context, userID, messageID, feedbackType string, recipients []string) error
}

// InboundEmailProcessor processes raw inbound emails from S3.
type InboundEmailProcessor interface {
	ProcessRawEmail(ctx context.Context, bucket, key string) error
}
