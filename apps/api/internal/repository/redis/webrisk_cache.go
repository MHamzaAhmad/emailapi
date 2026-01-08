package redis

import "context"

const webRiskBloomKey = "webrisk:bloom"

// WebRiskCache provides caching for Web Risk URL threat checking.
// It uses a Bloom filter internally to quickly identify potential threats.
type WebRiskCache struct {
	client *Client
}

// NewWebRiskCache creates a new WebRiskCache.
func NewWebRiskCache(client *Client) *WebRiskCache {
	return &WebRiskCache{client: client}
}

// CheckURLPrefixes checks if URL hash prefixes are potentially malicious.
// Returns true for each prefix that needs verification via Web Risk API.
// Uses a Bloom filter for fast lookups with possible false positives.
func (c *WebRiskCache) CheckURLPrefixes(ctx context.Context, hashPrefixes []string) ([]bool, error) {
	return c.client.BFMExists(ctx, webRiskBloomKey, hashPrefixes...)
}
