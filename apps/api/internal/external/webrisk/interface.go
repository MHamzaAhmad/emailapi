package webrisk

//go:generate mockgen -destination=mocks/mock_webrisk.go -package=mocks github.com/emailapi/api/internal/external/webrisk Client

import (
	"context"
)

// Client defines the interface for Web Risk API operations.
type Client interface {
	// Lookup verifies URLs against Web Risk Lookup API.
	// Called after Bloom filter indicates a potential match.
	Lookup(ctx context.Context, urls []string) ([]ThreatMatch, error)

	// FetchHashPrefixes gets hash prefixes from Web Risk Update API.
	// stateToken can be empty for initial fetch, or previous token for incremental updates.
	FetchHashPrefixes(ctx context.Context, stateToken string) (*UpdateResponse, error)
}
