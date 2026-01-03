package service

//go:generate mockgen -destination=mocks/mock_cache.go -package=mocks github.com/emailapi/api/internal/service Cache

import (
	rediscache "github.com/emailapi/api/internal/repository/redis"
)

// Cache aggregates all Redis cache access.
// This is injected into services for testability.
type Cache interface {
	Domain() rediscache.DomainCacheInterface
	APIKey() rediscache.APIKeyCacheInterface
	User() rediscache.UserCacheInterface
	MX() rediscache.MXCacheInterface
	Reputation() rediscache.ReputationCacheInterface
	Unsubscribe() rediscache.UnsubscribeCacheInterface
	PendingAttachment() rediscache.PendingAttachmentCacheInterface
}
