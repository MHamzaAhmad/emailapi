package redis

import "time"

// Default TTL values for caches
const (
	defaultCacheTTL = 5 * time.Minute
)

// CacheAggregator implements service.Cache using Redis.
type CacheAggregator struct {
	domain            *DomainCache
	apiKey            *APIKeyCache
	user              *UserCache
	mx                *MXCache
	reputation        *ReputationCache
	unsubscribe       *UnsubscribeCache
	pendingAttachment *PendingAttachmentCache
	credit            *CreditCache
	usage             *UsageCache
	webRisk           *WebRiskCache
	routing           *RoutingCache
}

// NewCacheAggregator creates a new CacheAggregator with all caches.
func NewCacheAggregator(client *Client) *CacheAggregator {
	return &CacheAggregator{
		domain:            NewDomainCache(client, defaultCacheTTL),
		apiKey:            NewAPIKeyCache(client, defaultCacheTTL),
		user:              NewUserCache(client, defaultCacheTTL),
		mx:                NewMXCache(client),
		reputation:        NewReputationCache(client),
		unsubscribe:       NewUnsubscribeCache(client),
		pendingAttachment: NewPendingAttachmentCache(client),
		credit:            NewCreditCache(client),
		usage:             NewUsageCache(client),
		webRisk:           NewWebRiskCache(client),
		routing:           NewRoutingCache(client.Underlying()),
	}
}

// Domain returns the domain cache.
func (c *CacheAggregator) Domain() DomainCacheInterface {
	return c.domain
}

// APIKey returns the API key cache.
func (c *CacheAggregator) APIKey() APIKeyCacheInterface {
	return c.apiKey
}

// User returns the user cache.
func (c *CacheAggregator) User() UserCacheInterface {
	return c.user
}

// MX returns the MX cache.
func (c *CacheAggregator) MX() MXCacheInterface {
	return c.mx
}

// Reputation returns the reputation cache.
func (c *CacheAggregator) Reputation() ReputationCacheInterface {
	return c.reputation
}

// Unsubscribe returns the unsubscribe cache.
func (c *CacheAggregator) Unsubscribe() UnsubscribeCacheInterface {
	return c.unsubscribe
}

// PendingAttachment returns the pending attachment cache.
func (c *CacheAggregator) PendingAttachment() PendingAttachmentCacheInterface {
	return c.pendingAttachment
}

// Credit returns the credit cache for Polar billing.
func (c *CacheAggregator) Credit() CreditCacheInterface {
	return c.credit
}

// Usage returns the usage cache for rate limiting and quotas.
func (c *CacheAggregator) Usage() UsageCacheInterface {
	return c.usage
}

// WebRisk returns the Web Risk cache for URL threat checking.
func (c *CacheAggregator) WebRisk() WebRiskCacheInterface {
	return c.webRisk
}

// Routing returns the routing cache for email reply threading.
func (c *CacheAggregator) Routing() RoutingCacheInterface {
	return c.routing
}
