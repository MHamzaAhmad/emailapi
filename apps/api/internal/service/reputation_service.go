package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository/suppression"
	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
	"github.com/emailapi/api/internal/worker"
)

// ReputationService handles user reputation business logic.
type ReputationService struct {
	store        Store
	riverClient  *river.Client[pgx.Tx]
	activityRepo *tbrepo.ActivityRepository
}

// NewReputationService creates a new ReputationService.
func NewReputationService(
	store Store,
	riverClient *river.Client[pgx.Tx],
	activityRepo *tbrepo.ActivityRepository,
) *ReputationService {
	return &ReputationService{
		store:        store,
		riverClient:  riverClient,
		activityRepo: activityRepo,
	}
}

// RecordBounceIncident records a bounce incident and queues evaluation.
func (s *ReputationService) RecordBounceIncident(
	ctx context.Context,
	userID string,
	messageID string,
	bounceType string,
	bounceSubType string,
	recipients []domain.BounceRecipient,
) error {
	incidentType := domain.IncidentTypeBounceSoft
	if bounceType == "Permanent" {
		incidentType = domain.IncidentTypeBounceHard
	}

	for _, recipient := range recipients {
		incident := &domain.ReputationIncident{
			ID:                 uuid.New().String(),
			UserID:             userID,
			IncidentType:       incidentType,
			MessageID:          messageID,
			RecipientEmailHash: suppression.HashEmail(recipient.EmailAddress),
			BounceType:         bounceType,
			BounceSubtype:      bounceSubType,
			DiagnosticCode:     recipient.DiagnosticCode,
		}

		if err := s.store.Reputation().InsertIncident(ctx, incident); err != nil {
			fmt.Printf("Warning: failed to record bounce incident: %v\n", err)
		}
	}

	return s.queueEvaluation(ctx, userID)
}

// RecordComplaintIncident records a complaint incident and queues evaluation.
func (s *ReputationService) RecordComplaintIncident(
	ctx context.Context,
	userID string,
	messageID string,
	feedbackType string,
	recipientEmails []string,
) error {
	for _, email := range recipientEmails {
		incident := &domain.ReputationIncident{
			ID:                    uuid.New().String(),
			UserID:                userID,
			IncidentType:          domain.IncidentTypeComplaint,
			MessageID:             messageID,
			RecipientEmailHash:    suppression.HashEmail(email),
			ComplaintFeedbackType: feedbackType,
		}

		if err := s.store.Reputation().InsertIncident(ctx, incident); err != nil {
			fmt.Printf("Warning: failed to record complaint incident: %v\n", err)
		}
	}

	return s.queueEvaluation(ctx, userID)
}

// queueEvaluation queues an async job to evaluate user reputation.
// Uses River's unique job feature to debounce rapid-fire incidents.
func (s *ReputationService) queueEvaluation(ctx context.Context, userID string) error {
	if s.riverClient == nil {
		return nil
	}

	_, err := s.riverClient.Insert(ctx, worker.EvaluateReputationArgs{
		UserID: userID,
	}, &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,
			ByPeriod: 30 * time.Second, // Debounce: only one eval per user per 30s
		},
	})
	return err
}

// GetUserReputation retrieves reputation stats for a user.
func (s *ReputationService) GetUserReputation(ctx context.Context, userID string) (*domain.UserReputation, error) {
	return s.store.Reputation().Get(ctx, userID)
}

// ListFlaggedUsers lists users flagged for review.
func (s *ReputationService) ListFlaggedUsers(ctx context.Context, limit, offset int) ([]*domain.UserReputation, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.Reputation().ListFlagged(ctx, limit, offset)
}

// SuspendUser manually suspends a user account.
func (s *ReputationService) SuspendUser(ctx context.Context, userID, suspendedBy, reason string) error {
	// Ensure reputation record exists
	if err := s.store.Reputation().EnsureExists(ctx, userID); err != nil {
		return fmt.Errorf("failed to ensure reputation record: %w", err)
	}

	if err := s.store.Reputation().Suspend(ctx, userID, suspendedBy, reason); err != nil {
		return fmt.Errorf("failed to suspend user: %w", err)
	}

	// Log activity
	if s.activityRepo != nil {
		s.activityRepo.Log(ctx, userID, "reputation", userID, "suspended", "success",
			fmt.Sprintf("Account suspended by %s: %s", suspendedBy, reason), nil)
	}
	return nil
}

// UnsuspendUser removes suspension from a user account.
func (s *ReputationService) UnsuspendUser(ctx context.Context, userID, unsuspendedBy string) error {
	if err := s.store.Reputation().Unsuspend(ctx, userID); err != nil {
		return fmt.Errorf("failed to unsuspend user: %w", err)
	}

	if s.activityRepo != nil {
		s.activityRepo.Log(ctx, userID, "reputation", userID, "unsuspended", "success",
			fmt.Sprintf("Account unsuspended by %s", unsuspendedBy), nil)
	}
	return nil
}

// ListIncidents retrieves incident history for a user.
func (s *ReputationService) ListIncidents(ctx context.Context, userID string, limit, offset int) ([]*domain.ReputationIncident, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.Reputation().ListIncidents(ctx, userID, limit, offset)
}

// IsUserSuspended checks if a user is currently suspended.
func (s *ReputationService) IsUserSuspended(ctx context.Context, userID string) (bool, error) {
	rep, err := s.store.Reputation().Get(ctx, userID)
	if err != nil {
		// No reputation record = not suspended
		return false, nil
	}
	return rep.IsSuspended, nil
}
