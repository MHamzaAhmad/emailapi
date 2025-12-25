package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/external/webrisk"
	"github.com/emailapi/api/internal/repository/redis"
)

const (
	// bloomFilterKey is the Redis key for the Web Risk hash prefix Bloom filter.
	bloomFilterKey = "webrisk:bloom"
	// stateTokenKey stores the last state token for incremental updates.
	stateTokenKey = "webrisk:state"
	// bloomErrorRate is the target false positive rate for the Bloom filter.
	bloomErrorRate = 0.001 // 0.1%
	// bloomCapacity is the expected number of items in the Bloom filter.
	bloomCapacity = 1000000 // 1 million entries
)

// WebRiskSyncArgs are the arguments for the Web Risk sync job.
type WebRiskSyncArgs struct{}

// Kind returns the job kind identifier.
func (WebRiskSyncArgs) Kind() string {
	return "webrisk_sync"
}

// InsertOpts returns the default insert options for the job.
func (WebRiskSyncArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: "default",
	}
}

// WebRiskSyncWorker syncs Web Risk hash prefixes to Redis Bloom filter.
type WebRiskSyncWorker struct {
	river.WorkerDefaults[WebRiskSyncArgs]
	redis   *redis.Client
	webrisk webrisk.Client
}

// NewWebRiskSyncWorker creates a new Web Risk sync worker.
func NewWebRiskSyncWorker(redisClient *redis.Client, webriskClient webrisk.Client) *WebRiskSyncWorker {
	return &WebRiskSyncWorker{
		redis:   redisClient,
		webrisk: webriskClient,
	}
}

// Work performs the Web Risk sync job.
func (w *WebRiskSyncWorker) Work(ctx context.Context, job *river.Job[WebRiskSyncArgs]) error {
	// Get previous state token for incremental update
	stateToken, _ := w.redis.Get(ctx, stateTokenKey)

	// Fetch hash prefixes from Web Risk Update API
	resp, err := w.webrisk.FetchHashPrefixes(ctx, stateToken)
	if err != nil {
		return fmt.Errorf("failed to fetch hash prefixes: %w", err)
	}

	if len(resp.HashPrefixes) == 0 {
		// No new prefixes to add
		return nil
	}

	// Convert hash prefixes to hex strings for Bloom filter
	hexPrefixes := make([]string, len(resp.HashPrefixes))
	for i, prefix := range resp.HashPrefixes {
		hexPrefixes[i] = fmt.Sprintf("%x", prefix)
	}

	// Add prefixes to Bloom filter in batches
	batchSize := 1000
	for i := 0; i < len(hexPrefixes); i += batchSize {
		end := i + batchSize
		if end > len(hexPrefixes) {
			end = len(hexPrefixes)
		}
		batch := hexPrefixes[i:end]

		if _, err := w.redis.BFMAdd(ctx, bloomFilterKey, batch...); err != nil {
			return fmt.Errorf("failed to add prefixes to bloom filter: %w", err)
		}
	}

	// Store state token for next incremental update
	if resp.StateToken != "" {
		if err := w.redis.Set(ctx, stateTokenKey, resp.StateToken, 0); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to store state token: %v\n", err)
		}
	}

	fmt.Printf("Synced %d hash prefixes to Bloom filter\n", len(hexPrefixes))
	return nil
}

// SchedulePeriodicSync schedules the Web Risk sync job to run periodically.
func SchedulePeriodicSync() *river.PeriodicJob {
	return river.NewPeriodicJob(
		river.PeriodicInterval(30*time.Minute),
		func() (river.JobArgs, *river.InsertOpts) {
			return WebRiskSyncArgs{}, nil
		},
		&river.PeriodicJobOpts{
			RunOnStart: true, // Run immediately on startup
		},
	)
}
