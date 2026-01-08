package redis

//go:generate mockgen -destination=mocks/mock_cache.go -package=mocks github.com/emailapi/api/internal/repository/redis DomainCacheInterface,APIKeyCacheInterface,UserCacheInterface,MXCacheInterface,ReputationCacheInterface,UnsubscribeCacheInterface,PendingAttachmentCacheInterface,WebRiskCacheInterface

import (
	"context"

	"github.com/emailapi/api/internal/domain"
)

// DomainCacheInterface defines the interface for domain caching operations.
type DomainCacheInterface interface {
	GetByID(ctx context.Context, id string) (*domain.SendingDomain, error)
	SetByID(ctx context.Context, d *domain.SendingDomain) error
	GetByUserID(ctx context.Context, userID string) ([]*domain.SendingDomain, error)
	SetByUserID(ctx context.Context, userID string, domains []*domain.SendingDomain) error
	InvalidateByID(ctx context.Context, id string) error
	InvalidateByUserID(ctx context.Context, userID string) error
	InvalidateAll(ctx context.Context, id, userID string) error

	// DomainWithDetails cache - preserves DNS record validation results
	// These are set after Verify() and used by Get()/List() to return validated statuses
	GetDetailsByID(ctx context.Context, id string) (*domain.DomainWithDetails, error)
	SetDetailsByID(ctx context.Context, d *domain.DomainWithDetails) error

	// Fast-path methods for sending validation (high-frequency, low-latency)
	GetSendingStatus(ctx context.Context, userID, domainName string) (*domain.SendingDomain, error)
	SetSendingStatus(ctx context.Context, d *domain.SendingDomain) error
	InvalidateSendingStatus(ctx context.Context, userID, domainName string) error
}

// APIKeyCacheInterface defines the interface for API key caching operations.
type APIKeyCacheInterface interface {
	GetByID(ctx context.Context, id string) (*domain.APIKey, error)
	SetByID(ctx context.Context, k *domain.APIKey) error
	GetByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error)
	SetByPrefix(ctx context.Context, k *domain.APIKey) error
	GetByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error)
	SetByUserID(ctx context.Context, userID string, keys []*domain.APIKey) error
	InvalidateByID(ctx context.Context, id string) error
	InvalidateByPrefix(ctx context.Context, prefix string) error
	InvalidateByUserID(ctx context.Context, userID string) error
	InvalidateAll(ctx context.Context, id, prefix, userID string) error

	// Fast-path methods for auth validation (high-frequency, low-latency)
	GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error)
	SetByKeyHash(ctx context.Context, keyHash string, k *domain.APIKey) error
	InvalidateByKeyHash(ctx context.Context, keyHash string) error
}

// UserCacheInterface defines the interface for user caching operations.
type UserCacheInterface interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	SetByID(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	SetByEmail(ctx context.Context, u *domain.User) error
	InvalidateByID(ctx context.Context, id string) error
	InvalidateByEmail(ctx context.Context, email string) error
	InvalidateAll(ctx context.Context, id, email string) error

	// External ID lookup cache (for Clerk JWT authentication)
	GetByExternalID(ctx context.Context, externalID string) (*domain.User, error)
	SetByExternalID(ctx context.Context, u *domain.User) error
	InvalidateByExternalID(ctx context.Context, externalID string) error
}

// ReputationCacheInterface defines the interface for reputation caching.
type ReputationCacheInterface interface {
	Get(ctx context.Context, userID string) (*UserReputationStatus, error)
	Set(ctx context.Context, userID string, status *UserReputationStatus) error
	SetSuspended(ctx context.Context, userID string) error
	Delete(ctx context.Context, userID string) error
}

// UnsubscribeCacheInterface defines the interface for unsubscribe caching.
type UnsubscribeCacheInterface interface {
	Set(ctx context.Context, userID, emailHash string) error
	Check(ctx context.Context, userID, emailHash string) (bool, error)
	CheckBatch(ctx context.Context, userID string, hashes []string) ([]string, error)
	SetBatch(ctx context.Context, userID string, hashes []string) error
	Delete(ctx context.Context, userID, emailHash string) error
}

// PendingAttachmentCacheInterface defines the interface for pending attachment caching.
// Used for event-driven attachment scanning with GuardDuty.
type PendingAttachmentCacheInterface interface {
	Store(ctx context.Context, data *PendingAttachmentData) error
	GetByEmailID(ctx context.Context, emailID string) (*PendingAttachmentData, error)
	GetByS3Key(ctx context.Context, s3Key string) (*PendingAttachmentData, error)
	Delete(ctx context.Context, emailID string) error
	MarkAttachmentScanned(ctx context.Context, s3Key string) (*PendingAttachmentData, bool, error)
}

// WebRiskCacheInterface defines the interface for Web Risk URL threat checking.
type WebRiskCacheInterface interface {
	// CheckURLPrefixes checks if URL hash prefixes are potentially malicious.
	// Returns true for each prefix that needs verification via Web Risk API.
	CheckURLPrefixes(ctx context.Context, hashPrefixes []string) ([]bool, error)
}

// Ensure concrete types implement interfaces
var _ DomainCacheInterface = (*DomainCache)(nil)
var _ APIKeyCacheInterface = (*APIKeyCache)(nil)
var _ UserCacheInterface = (*UserCache)(nil)
var _ ReputationCacheInterface = (*ReputationCache)(nil)
var _ UnsubscribeCacheInterface = (*UnsubscribeCache)(nil)
var _ PendingAttachmentCacheInterface = (*PendingAttachmentCache)(nil)
var _ WebRiskCacheInterface = (*WebRiskCache)(nil)
