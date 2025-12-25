package webrisk

import (
	"context"
)

// ThreatType represents the type of threat detected.
type ThreatType string

const (
	ThreatTypeMalware           ThreatType = "MALWARE"
	ThreatTypeSocialEngineering ThreatType = "SOCIAL_ENGINEERING"
	ThreatTypeUnwantedSoftware  ThreatType = "UNWANTED_SOFTWARE"
)

// ThreatMatch represents a URL that matched a threat.
type ThreatMatch struct {
	URL        string
	ThreatType ThreatType
}

// UpdateResponse contains hash prefixes from the Update API.
type UpdateResponse struct {
	// HashPrefixes are 4-byte SHA256 prefixes of threat URLs.
	HashPrefixes [][]byte
	// StateToken for incremental updates.
	StateToken string
	// NegativeCacheDuration in seconds before next update.
	NegativeCacheDuration int64
}

// Client defines the interface for Web Risk API operations.
type Client interface {
	// Lookup verifies URLs against Web Risk Lookup API.
	// Called after Bloom filter indicates a potential match.
	Lookup(ctx context.Context, urls []string) ([]ThreatMatch, error)

	// FetchHashPrefixes gets hash prefixes from Web Risk Update API.
	// stateToken can be empty for initial fetch, or previous token for incremental updates.
	FetchHashPrefixes(ctx context.Context, stateToken string) (*UpdateResponse, error)
}
