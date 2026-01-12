package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/domain"
	rediscache "github.com/emailapi/api/internal/repository/redis"
	"github.com/emailapi/api/internal/repository/suppression"
	"github.com/emailapi/api/internal/worker"
)

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
// Returns ErrAccountSuspended if user is hard suspended.
func (s *ReputationService) CheckSendPermission(ctx context.Context, userID string) error {
	// 1. Try Redis cache first (fast path)
	if s.cache != nil {
		status, err := s.cache.Reputation().Get(ctx, userID)
		if err == nil && status != nil {
			// Hard suspended = blocked completely
			if status.IsHardSuspended || status.IsSuspended {
				return domain.ErrAccountSuspended
			}
			// Soft suspended = can send but with reduced limits
			return nil
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
			IsFlagged:       rep.IsFlagged,
			IsSuspended:     rep.IsSuspended,
			IsHardSuspended: rep.IsSuspended, // Legacy IsSuspended = hard
			IsSoftSuspended: rep.IsFlagged && !rep.IsSuspended,
			Score:           rep.SuspensionScore,
		})
	}

	if rep.IsSuspended {
		return domain.ErrAccountSuspended
	}
	return nil
}

// GetEffectiveRateLimit returns the effective rate limit for a user.
// Soft suspended/flagged users get reduced limits (10% of normal).
func (s *ReputationService) GetEffectiveRateLimit(ctx context.Context, userID string, baseLimit int) int {
	// Check cache first
	if s.cache != nil {
		status, err := s.cache.Reputation().Get(ctx, userID)
		if err == nil && status != nil {
			// Soft suspended or flagged = 10% limit
			if status.IsSoftSuspended || status.IsFlagged {
				reduced := baseLimit / 10
				if reduced < 1 {
					reduced = 1
				}
				return reduced
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
		reduced := baseLimit / 10
		if reduced < 1 {
			reduced = 1
		}
		return reduced
	}
	return baseLimit
}

// GetSuspensionStatus returns detailed suspension status for a user.
// Used by the limit engine for comprehensive checking.
func (s *ReputationService) GetSuspensionStatus(ctx context.Context, userID string) (isHardSuspended, isSoftSuspended bool, err error) {
	// Check cache first
	if s.cache != nil {
		status, cacheErr := s.cache.Reputation().Get(ctx, userID)
		if cacheErr == nil && status != nil {
			return status.IsHardSuspended || status.IsSuspended,
				status.IsSoftSuspended || status.IsFlagged,
				nil
		}
	}

	// Cache miss: check database
	rep, dbErr := s.store.Reputation().Get(ctx, userID)
	if dbErr != nil {
		// No record = not suspended
		return false, false, nil
	}

	return rep.IsSuspended, rep.IsFlagged && !rep.IsSuspended, nil
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

// SuspendUser manually suspends a user account (hard suspension).
// Deprecated: Use HardSuspendUser or SoftSuspendUser instead.
func (s *ReputationService) SuspendUser(ctx context.Context, userID, suspendedBy, reason string) error {
	return s.HardSuspendUser(ctx, userID, suspendedBy, reason)
}

// HardSuspendUser blocks a user from sending any emails.
func (s *ReputationService) HardSuspendUser(ctx context.Context, userID, suspendedBy, reason string) error {
	// Ensure reputation record exists
	if err := s.store.Reputation().EnsureExists(ctx, userID); err != nil {
		return domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "ensure_reputation")
	}

	if err := s.store.Reputation().Suspend(ctx, userID, suspendedBy, reason); err != nil {
		return domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "suspend_user")
	}

	// Update cache to immediately block sends
	if s.cache != nil {
		s.cache.Reputation().SetHardSuspended(ctx, userID)
	}

	// Log activity
	if s.analytics != nil {
		s.analytics.Activity().Log(ctx, userID, "reputation", userID, "hard_suspended", "success",
			fmt.Sprintf("Account hard suspended by %s: %s", suspendedBy, reason), nil)
	}
	return nil
}

// SoftSuspendUser puts a user on restricted mode (reduced limits but can still send).
func (s *ReputationService) SoftSuspendUser(ctx context.Context, userID, suspendedBy, reason string) error {
	// Ensure reputation record exists
	if err := s.store.Reputation().EnsureExists(ctx, userID); err != nil {
		return domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "ensure_reputation")
	}

	// Get current stats to preserve them
	rep, err := s.store.Reputation().Get(ctx, userID)
	if err != nil {
		// No existing record, create minimal one
		rep = &domain.UserReputation{UserID: userID}
	}

	// Update to flagged state (soft suspension)
	rep.IsFlagged = true
	rep.FlaggedReason = reason
	if err := s.store.Reputation().UpdateStats(ctx, userID, rep); err != nil {
		return domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "soft_suspend_user")
	}

	// Update cache with soft suspension
	if s.cache != nil {
		s.cache.Reputation().SetSoftSuspended(ctx, userID)
	}

	// Log activity
	if s.analytics != nil {
		s.analytics.Activity().Log(ctx, userID, "reputation", userID, "soft_suspended", "success",
			fmt.Sprintf("Account soft suspended by %s: %s", suspendedBy, reason), nil)
	}
	return nil
}

// UnsuspendUser removes suspension from a user account.
func (s *ReputationService) UnsuspendUser(ctx context.Context, userID, unsuspendedBy string) error {
	if err := s.store.Reputation().Unsuspend(ctx, userID); err != nil {
		return domain.ErrInternal.Clone().WithCause(err).WithMeta("operation", "unsuspend_user")
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
