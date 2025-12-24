package suppression

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/emailapi/api/internal/db"
	"github.com/emailapi/api/internal/repository/redis"
)

// Reason represents why an email was suppressed.
type Reason string

const (
	ReasonBounceHard Reason = "bounce_hard" // Permanent - no TTL
	ReasonBounceSoft Reason = "bounce_soft" // 72 hours TTL
	ReasonComplaint  Reason = "complaint"   // Permanent - no TTL
)

const (
	// Redis key prefix for suppression entries
	keyPrefix = "suppression:"

	// TTL for soft bounces (transient issues)
	softBounceTTL = 72 * time.Hour
)

// Entry represents a suppression list entry.
type Entry struct {
	EmailHash       string
	UserID          string
	Reason          Reason
	BounceType      string
	SourceMessageID string
	ExpiresAt       *time.Time
	CreatedAt       time.Time
}

// Repository provides hybrid Redis + PostgreSQL suppression operations.
type Repository struct {
	redis   *redis.Client
	queries *db.Queries
}

// NewRepository creates a new suppression repository.
func NewRepository(redisClient *redis.Client, queries *db.Queries) *Repository {
	return &Repository{
		redis:   redisClient,
		queries: queries,
	}
}

// HashEmail creates a SHA-256 hash of a lowercased, trimmed email address.
func HashEmail(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}

// Add stores a suppression entry in both Redis and PostgreSQL.
func (r *Repository) Add(ctx context.Context, entry *Entry) error {
	// Determine TTL based on reason
	var ttl time.Duration
	var expiresAt pgtype.Timestamptz

	if entry.Reason == ReasonBounceSoft {
		ttl = softBounceTTL
		expTime := time.Now().Add(ttl)
		entry.ExpiresAt = &expTime
		expiresAt = pgtype.Timestamptz{Time: expTime, Valid: true}
	}
	// Hard bounces and complaints: no TTL (permanent)

	// Write to Redis (primary for reads)
	redisKey := keyPrefix + entry.EmailHash
	redisValue := string(entry.Reason)
	if err := r.redis.Set(ctx, redisKey, redisValue, ttl); err != nil {
		// Log but don't fail - PostgreSQL is the backup
		fmt.Printf("Warning: failed to write suppression to Redis: %v\n", err)
	}

	// Write to PostgreSQL (backup/durability)
	params := db.InsertSuppressionParams{
		EmailHash: entry.EmailHash,
		UserID:    pgtype.Text{String: entry.UserID, Valid: entry.UserID != ""},
		Reason:    string(entry.Reason),
		ExpiresAt: expiresAt,
	}
	if entry.BounceType != "" {
		params.BounceType = pgtype.Text{String: entry.BounceType, Valid: true}
	}
	if entry.SourceMessageID != "" {
		params.SourceMessageID = pgtype.Text{String: entry.SourceMessageID, Valid: true}
	}

	if err := r.queries.InsertSuppression(ctx, params); err != nil {
		return fmt.Errorf("failed to insert suppression: %w", err)
	}

	return nil
}

// CheckBatch checks multiple email hashes for suppression using Redis MGET.
// Returns the list of hashes that are suppressed.
func (r *Repository) CheckBatch(ctx context.Context, hashes []string) ([]string, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	// Build Redis keys
	keys := make([]string, len(hashes))
	for i, hash := range hashes {
		keys[i] = keyPrefix + hash
	}

	// Single MGET call for all hashes
	values, err := r.redis.MGet(ctx, keys...)
	if err != nil {
		return nil, fmt.Errorf("redis mget failed: %w", err)
	}

	// Collect suppressed hashes
	var suppressed []string
	for i, val := range values {
		if val != nil {
			suppressed = append(suppressed, hashes[i])
		}
	}

	return suppressed, nil
}

// Remove deletes a suppression entry from both Redis and PostgreSQL.
func (r *Repository) Remove(ctx context.Context, emailHash string) error {
	// Remove from Redis
	redisKey := keyPrefix + emailHash
	if err := r.redis.Del(ctx, redisKey); err != nil {
		fmt.Printf("Warning: failed to delete suppression from Redis: %v\n", err)
	}

	// Remove from PostgreSQL
	if err := r.queries.DeleteSuppressionByHash(ctx, emailHash); err != nil {
		return fmt.Errorf("failed to delete suppression: %w", err)
	}

	return nil
}

// SyncFromPostgres reloads all active suppressions from PostgreSQL into Redis.
// This is called on startup to recover from Redis restarts.
func (r *Repository) SyncFromPostgres(ctx context.Context) error {
	entries, err := r.queries.ListActiveSuppressions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active suppressions: %w", err)
	}

	synced := 0
	for _, entry := range entries {
		redisKey := keyPrefix + entry.EmailHash

		// Calculate remaining TTL if expires_at is set
		var ttl time.Duration
		if entry.ExpiresAt.Valid {
			remaining := time.Until(entry.ExpiresAt.Time)
			if remaining <= 0 {
				// Already expired, skip
				continue
			}
			ttl = remaining
		}
		// No TTL for permanent entries

		if err := r.redis.Set(ctx, redisKey, entry.Reason, ttl); err != nil {
			fmt.Printf("Warning: failed to sync suppression %s to Redis: %v\n", entry.EmailHash, err)
			continue
		}
		synced++
	}

	fmt.Printf("Synced %d suppression entries from PostgreSQL to Redis\n", synced)
	return nil
}

// Cleanup deletes expired entries from PostgreSQL.
// Redis handles its own TTL expiration automatically.
func (r *Repository) Cleanup(ctx context.Context) (int64, error) {
	deleted, err := r.queries.DeleteExpiredSuppressions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired suppressions: %w", err)
	}
	return deleted, nil
}
