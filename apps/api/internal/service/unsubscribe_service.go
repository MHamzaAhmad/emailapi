package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/emailapi/api/internal/domain"
	"github.com/emailapi/api/internal/repository/postgres"
)

// UnsubscribeService handles unsubscribe operations with cache-first strategy.
type UnsubscribeService struct {
	repo     postgres.UnsubscribeRepository
	cache    Cache
	tokenSvc *UnsubscribeTokenService
	baseURL  string
}

// NewUnsubscribeService creates a new UnsubscribeService.
func NewUnsubscribeService(
	repo postgres.UnsubscribeRepository,
	cache Cache,
	tokenSvc *UnsubscribeTokenService,
	baseURL string,
) *UnsubscribeService {
	return &UnsubscribeService{
		repo:     repo,
		cache:    cache,
		tokenSvc: tokenSvc,
		baseURL:  strings.TrimSuffix(baseURL, "/"),
	}
}

// HashEmail creates a SHA-256 hash of a lowercased, trimmed email address.
func HashEmail(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}

// GenerateLink creates an unsubscribe URL for a recipient.
func (s *UnsubscribeService) GenerateLink(userID, recipientEmail, emailID string) (string, error) {
	return s.tokenSvc.GenerateLink(s.baseURL, userID, recipientEmail, emailID)
}

// ProcessUnsubscribe adds an email to the unsubscribe list after token verification.
func (s *UnsubscribeService) ProcessUnsubscribe(ctx context.Context, tokenStr string, source domain.UnsubscribeSource) (*UnsubscribeTokenData, error) {
	// Decode and verify token
	tokenData, err := s.tokenSvc.Decode(tokenStr)
	if err != nil {
		return nil, err
	}

	emailHash := HashEmail(tokenData.Email)

	// Create entry
	entry := &domain.UnsubscribeEntry{
		ID:            uuid.New().String(),
		UserID:        tokenData.UserID,
		EmailHash:     emailHash,
		SourceEmailID: tokenData.EmailID,
		Source:        source,
	}

	// Write to DB first (source of truth)
	if err := s.repo.Add(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to add unsubscribe: %w", err)
	}

	// Update cache (best-effort, don't fail if cache fails)
	if s.cache != nil {
		if err := s.cache.Unsubscribe().Set(ctx, tokenData.UserID, emailHash); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to update unsubscribe cache: %v\n", err)
		}
	}

	return tokenData, nil
}

// CheckBatch checks if any recipients are unsubscribed for a user.
// Uses cache-first strategy: check Redis first, fallback to PostgreSQL for misses.
// Returns the list of email addresses that are unsubscribed.
func (s *UnsubscribeService) CheckBatch(ctx context.Context, userID string, emails []string) ([]string, error) {
	if len(emails) == 0 {
		return nil, nil
	}

	// Convert emails to hashes
	hashToEmail := make(map[string]string, len(emails))
	hashes := make([]string, len(emails))
	for i, email := range emails {
		hash := HashEmail(email)
		hashes[i] = hash
		hashToEmail[hash] = email
	}

	var unsubscribedHashes []string

	// Step 1: Check cache first
	if s.cache != nil {
		cachedHashes, err := s.cache.Unsubscribe().CheckBatch(ctx, userID, hashes)
		if err != nil {
			// Log but fallback to DB
			fmt.Printf("Warning: cache check failed, falling back to DB: %v\n", err)
		} else {
			unsubscribedHashes = cachedHashes

			// If all found in cache, return early
			if len(cachedHashes) == len(hashes) {
				return hashesToEmails(cachedHashes, hashToEmail), nil
			}

			// Remove found hashes for DB lookup
			foundSet := make(map[string]bool, len(cachedHashes))
			for _, h := range cachedHashes {
				foundSet[h] = true
			}

			var remainingHashes []string
			for _, h := range hashes {
				if !foundSet[h] {
					remainingHashes = append(remainingHashes, h)
				}
			}
			hashes = remainingHashes
		}
	}

	// Step 2: Check DB for remaining hashes
	if len(hashes) > 0 {
		dbHashes, err := s.repo.CheckBatch(ctx, userID, hashes)
		if err != nil {
			return nil, fmt.Errorf("failed to check unsubscribes: %w", err)
		}

		// Populate cache for DB hits
		if s.cache != nil && len(dbHashes) > 0 {
			if err := s.cache.Unsubscribe().SetBatch(ctx, userID, dbHashes); err != nil {
				fmt.Printf("Warning: failed to populate unsubscribe cache: %v\n", err)
			}
		}

		unsubscribedHashes = append(unsubscribedHashes, dbHashes...)
	}

	return hashesToEmails(unsubscribedHashes, hashToEmail), nil
}

// Resubscribe removes an email from the unsubscribe list.
func (s *UnsubscribeService) Resubscribe(ctx context.Context, userID, email string) error {
	emailHash := HashEmail(email)

	// Remove from DB
	if err := s.repo.Delete(ctx, userID, emailHash); err != nil {
		return fmt.Errorf("failed to delete unsubscribe: %w", err)
	}

	// Remove from cache
	if s.cache != nil {
		if err := s.cache.Unsubscribe().Delete(ctx, userID, emailHash); err != nil {
			fmt.Printf("Warning: failed to delete from unsubscribe cache: %v\n", err)
		}
	}

	return nil
}

// SyncCache syncs all unsubscribes from PostgreSQL to Redis.
// Should be called on startup in a background goroutine.
func (s *UnsubscribeService) SyncCache(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}

	entries, err := s.repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to list unsubscribes: %w", err)
	}

	// Group by user for batch operations
	byUser := make(map[string][]string)
	for _, entry := range entries {
		byUser[entry.UserID] = append(byUser[entry.UserID], entry.EmailHash)
	}

	synced := 0
	for userID, hashes := range byUser {
		if err := s.cache.Unsubscribe().SetBatch(ctx, userID, hashes); err != nil {
			fmt.Printf("Warning: failed to sync unsubscribes for user %s: %v\n", userID, err)
			continue
		}
		synced += len(hashes)
	}

	fmt.Printf("Synced %d unsubscribe entries from PostgreSQL to Redis\n", synced)
	return nil
}

// TokenService returns the token service for external use (HTTP handler).
func (s *UnsubscribeService) TokenService() *UnsubscribeTokenService {
	return s.tokenSvc
}

// BaseURL returns the unsubscribe base URL for worker job configuration.
func (s *UnsubscribeService) BaseURL() string {
	return s.baseURL
}

// TokenSecret returns the token secret for worker job configuration.
func (s *UnsubscribeService) TokenSecret() string {
	if s.tokenSvc == nil {
		return ""
	}
	return string(s.tokenSvc.secret)
}

// hashesToEmails converts a list of hashes back to emails using the lookup map.
func hashesToEmails(hashes []string, hashToEmail map[string]string) []string {
	if len(hashes) == 0 {
		return nil
	}
	emails := make([]string, 0, len(hashes))
	for _, h := range hashes {
		if email, ok := hashToEmail[h]; ok {
			emails = append(emails, email)
		}
	}
	return emails
}
