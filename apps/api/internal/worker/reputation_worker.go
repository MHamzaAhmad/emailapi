package worker

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository/postgres"
	redisrepo "github.com/emailapi/api/internal/repository/redis"
	tbrepo "github.com/emailapi/api/internal/repository/tinybird"
)

// EvaluateReputationArgs contains the job arguments for reputation evaluation.
type EvaluateReputationArgs struct {
	UserID string `json:"user_id"`
}

// Kind returns the job kind identifier.
func (EvaluateReputationArgs) Kind() string { return "evaluate_reputation" }

// ReputationWorker processes reputation evaluation jobs.
type ReputationWorker struct {
	river.WorkerDefaults[EvaluateReputationArgs]
	reputationRepo postgres.ReputationRepository
	activityRepo   *tbrepo.ActivityRepository
	cache          *redisrepo.ReputationCache
}

// NewReputationWorker creates a new ReputationWorker.
func NewReputationWorker(
	reputationRepo postgres.ReputationRepository,
	activityRepo *tbrepo.ActivityRepository,
	cache *redisrepo.ReputationCache,
) *ReputationWorker {
	return &ReputationWorker{
		reputationRepo: reputationRepo,
		activityRepo:   activityRepo,
		cache:          cache,
	}
}

// Thresholds for flagging and auto-suspension
const (
	// Flagging thresholds
	FlagScoreThreshold   = 50.0
	FlagComplaints30dMin = 3
	FlagHardBouncesMin   = 10

	// Auto-suspension thresholds (severe violations)
	SuspendScoreThreshold   = 100.0
	SuspendComplaints30dMin = 5
	SuspendHardBouncesMin   = 20
)

// Work evaluates user reputation based on incident history.
func (w *ReputationWorker) Work(ctx context.Context, job *river.Job[EvaluateReputationArgs]) error {
	userID := job.Args.UserID

	// 1. Ensure user_reputation record exists
	if err := w.reputationRepo.EnsureExists(ctx, userID); err != nil {
		return fmt.Errorf("failed to ensure reputation record: %w", err)
	}

	// 2. Count all incidents
	stats, err := w.reputationRepo.CountIncidents(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to count incidents: %w", err)
	}

	// 3. Calculate suspension score
	// Weights: hard_bounce=5, soft_bounce=1, complaint=10
	lifetimeScore := float64(stats.HardBounces*5 + stats.SoftBounces + stats.Complaints*10)
	recentScore := float64(stats.Bounces30d*3 + stats.Complaints30d*10)

	// Combined score (weighted toward recent activity)
	finalScore := (lifetimeScore * 0.3) + (recentScore * 0.7)

	// 4. Determine if user should be flagged
	shouldFlag := finalScore > FlagScoreThreshold ||
		stats.Complaints30d >= FlagComplaints30dMin ||
		stats.HardBounces >= FlagHardBouncesMin

	flagReason := ""
	if shouldFlag {
		if stats.Complaints30d >= FlagComplaints30dMin {
			flagReason = fmt.Sprintf("High complaint rate: %d complaints in 30 days", stats.Complaints30d)
		} else if stats.HardBounces >= FlagHardBouncesMin {
			flagReason = fmt.Sprintf("High hard bounce count: %d total", stats.HardBounces)
		} else {
			flagReason = fmt.Sprintf("High suspension score: %.2f", finalScore)
		}
	}

	// 5. Determine if user should be auto-suspended (severe violations)
	shouldAutoSuspend := finalScore > SuspendScoreThreshold ||
		stats.Complaints30d >= SuspendComplaints30dMin ||
		stats.HardBounces >= SuspendHardBouncesMin

	suspendReason := ""
	if shouldAutoSuspend {
		if stats.Complaints30d >= SuspendComplaints30dMin {
			suspendReason = fmt.Sprintf("Auto-suspended: %d complaints in 30 days exceeds threshold", stats.Complaints30d)
		} else if stats.HardBounces >= SuspendHardBouncesMin {
			suspendReason = fmt.Sprintf("Auto-suspended: %d hard bounces exceeds threshold", stats.HardBounces)
		} else {
			suspendReason = fmt.Sprintf("Auto-suspended: critical reputation score %.2f", finalScore)
		}
	}

	// 6. Get current reputation to check if newly flagged/suspended
	currentRep, _ := w.reputationRepo.Get(ctx, userID)
	wasAlreadyFlagged := currentRep != nil && currentRep.IsFlagged
	wasAlreadySuspended := currentRep != nil && currentRep.IsSuspended

	// 7. Update reputation stats
	if err := w.reputationRepo.UpdateStats(ctx, userID, &domain.UserReputation{
		TotalBounces:    stats.HardBounces + stats.SoftBounces,
		HardBounces:     stats.HardBounces,
		SoftBounces:     stats.SoftBounces,
		Complaints:      stats.Complaints,
		Bounces30d:      stats.Bounces30d,
		Complaints30d:   stats.Complaints30d,
		SuspensionScore: finalScore,
		IsFlagged:       shouldFlag,
		FlaggedReason:   flagReason,
	}); err != nil {
		return fmt.Errorf("failed to update reputation stats: %w", err)
	}

	// 8. Handle auto-suspension if triggered
	if shouldAutoSuspend && !wasAlreadySuspended {
		if err := w.reputationRepo.Suspend(ctx, userID, "system", suspendReason); err != nil {
			return fmt.Errorf("failed to auto-suspend user: %w", err)
		}

		// Log auto-suspension as critical event
		if w.activityRepo != nil {
			w.activityRepo.Log(ctx, userID, "reputation", userID, "auto_suspended", "critical",
				suspendReason, map[string]interface{}{
					"score":          finalScore,
					"hard_bounces":   stats.HardBounces,
					"complaints_30d": stats.Complaints30d,
				})
		}

		// Immediately update cache to block sends
		if w.cache != nil {
			w.cache.SetSuspended(ctx, userID)
		}

		return nil // Don't log flagging if we auto-suspended
	}

	// 9. Log if newly flagged (but not suspended)
	if shouldFlag && !wasAlreadyFlagged && w.activityRepo != nil {
		w.activityRepo.Log(ctx, userID, "reputation", userID, "flagged", "warning",
			flagReason, map[string]interface{}{
				"score":          finalScore,
				"hard_bounces":   stats.HardBounces,
				"complaints_30d": stats.Complaints30d,
			})
	}

	// 10. Invalidate cache to reflect new status
	if w.cache != nil {
		w.cache.Delete(ctx, userID)
	}

	return nil
}
