package worker

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository"
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
	reputationRepo repository.ReputationRepository
	activityRepo   *tbrepo.ActivityRepository
}

// NewReputationWorker creates a new ReputationWorker.
func NewReputationWorker(
	reputationRepo repository.ReputationRepository,
	activityRepo *tbrepo.ActivityRepository,
) *ReputationWorker {
	return &ReputationWorker{
		reputationRepo: reputationRepo,
		activityRepo:   activityRepo,
	}
}

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
	shouldFlag := finalScore > 50 || stats.Complaints30d >= 3 || stats.HardBounces >= 10
	flagReason := ""
	if shouldFlag {
		if stats.Complaints30d >= 3 {
			flagReason = fmt.Sprintf("High complaint rate: %d complaints in 30 days", stats.Complaints30d)
		} else if stats.HardBounces >= 10 {
			flagReason = fmt.Sprintf("High hard bounce count: %d total", stats.HardBounces)
		} else {
			flagReason = fmt.Sprintf("High suspension score: %.2f", finalScore)
		}
	}

	// 5. Get current reputation to check if newly flagged
	currentRep, _ := w.reputationRepo.Get(ctx, userID)
	wasAlreadyFlagged := currentRep != nil && currentRep.IsFlagged

	// 6. Update reputation stats
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

	// 7. Log if newly flagged
	if shouldFlag && !wasAlreadyFlagged && w.activityRepo != nil {
		w.activityRepo.Log(ctx, userID, "reputation", userID, "flagged", "warning",
			flagReason, map[string]interface{}{
				"score":          finalScore,
				"hard_bounces":   stats.HardBounces,
				"complaints_30d": stats.Complaints30d,
			})
	}

	return nil
}
