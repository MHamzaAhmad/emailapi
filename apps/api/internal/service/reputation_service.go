package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/domain"
	rediscache "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/worker"
)

// ErrAccountSuspended is returned when a suspended user tries to send email.
var ErrAccountSuspended = errors.New("account suspended: please contact support")

// ReputationService handles user reputation business logic.
// Implements ReputationChecker interface for use in email validation.
// ReputationService handles user reputation business logic.
// Implements ReputationChecker interface for use in email validation.
type ReputationService struct {
	store     Store
	queue     QueueClient
	cache     Cache
	analytics Analytics
}

// NewReputationService creates a new ReputationService.
func NewReputationService(
	store Store,
	queue QueueClient,
	cache Cache,
	analytics Analytics,
) *ReputationService {
	return &ReputationService{
		store:     store,
		queue:     queue,
		cache:     cache,
		analytics: analytics,
	}
}

// CheckSendPermission checks if a user is allowed to send emails.
// Uses Redis cache for O(1) lookups in the hot path.
// Returns ErrAccountSuspended if user is suspended.
func (s *ReputationService) CheckSendPermission(ctx context.Context, userID string) error {
	// 1. Try Redis cache first (fast path)
	if s.cache != nil {
		status, err := s.cache.Reputation().Get(ctx, userID)
		if err == nil && status != nil {
			if status.IsSuspended {
				return ErrAccountSuspended
			}
			return nil // Flagged users can send (with reduced limits)
		}
	}

	// 2. Cache miss: check database
	rep, err := s.store.Reputation().Get(ctx, userID)
	if err != nil {
		// No reputation record = not suspended, user is clean
		return nil
	}

	// 3. Update cache for future lookups
	if s.cache != nil {
		s.cache.Reputation().Set(ctx, userID, &rediscache.UserReputationStatus{
			IsFlagged:   rep.IsFlagged,
			IsSuspended: rep.IsSuspended,
			Score:       rep.SuspensionScore,
		})
	}

	if rep.IsSuspended {
		return ErrAccountSuspended
	}
	return nil
}

// GetEffectiveRateLimit returns the effective rate limit for a user.
// Flagged users get reduced limits (10% of normal).
func (s *ReputationService) GetEffectiveRateLimit(ctx context.Context, userID string, baseLimit int) int {
	// Check cache first
	if s.cache != nil {
		status, err := s.cache.Reputation().Get(ctx, userID)
		if err == nil && status != nil {
			if status.IsFlagged {
				return baseLimit / 10 // 10% of normal limit
			}
			return baseLimit
		}
	}

	// Cache miss: check database
	rep, err := s.store.Reputation().Get(ctx, userID)
	if err != nil {
		return baseLimit // No record = full limit
	}

	if rep.IsFlagged {
		return baseLimit / 10 // 10% of normal limit
	}
	return baseLimit
}

// InvalidateCache removes the cached reputation status for a user.
// Should be called after suspend/unsuspend/flag changes.
func (s *ReputationService) InvalidateCache(ctx context.Context, userID string) error {
	if s.cache != nil {
		return s.cache.Reputation().Delete(ctx, userID)
	}
	return nil
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

	// Log activity for bounce incident
	if s.analytics != nil {
		s.analytics.Activity().Log(ctx, userID, "reputation", messageID, "bounce_incident", "info",
			fmt.Sprintf("%s bounce: %d recipients", bounceType, len(recipients)),
			map[string]interface{}{
				"bounce_type":     bounceType,
				"bounce_subtype":  bounceSubType,
				"recipient_count": len(recipients),
			})
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

	// Log activity for complaint incident
	if s.analytics != nil {
		s.analytics.Activity().Log(ctx, userID, "reputation", messageID, "complaint_incident", "warning",
			fmt.Sprintf("Complaint: %s - %d recipients", feedbackType, len(recipientEmails)),
			map[string]interface{}{
				"feedback_type":   feedbackType,
				"recipient_count": len(recipientEmails),
			})
	}

	return s.queueEvaluation(ctx, userID)
}

// queueEvaluation queues an async job to evaluate user reputation.
// Uses River's unique job feature to debounce rapid-fire incidents.
func (s *ReputationService) queueEvaluation(ctx context.Context, userID string) error {
	if s.queue == nil {
		return nil
	}

	_, err := s.queue.Insert(ctx, worker.EvaluateReputationArgs{
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

	// Invalidate cache to immediately block sends
	s.InvalidateCache(ctx, userID)

	// Log activity
	if s.analytics != nil {
		s.analytics.Activity().Log(ctx, userID, "reputation", userID, "suspended", "success",
			fmt.Sprintf("Account suspended by %s: %s", suspendedBy, reason), nil)
	}
	return nil
}

// UnsuspendUser removes suspension from a user account.
func (s *ReputationService) UnsuspendUser(ctx context.Context, userID, unsuspendedBy string) error {
	if err := s.store.Reputation().Unsuspend(ctx, userID); err != nil {
		return fmt.Errorf("failed to unsuspend user: %w", err)
	}

	// Invalidate cache to allow sends
	s.InvalidateCache(ctx, userID)

	if s.analytics != nil {
		s.analytics.Activity().Log(ctx, userID, "reputation", userID, "unsuspended", "success",
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
