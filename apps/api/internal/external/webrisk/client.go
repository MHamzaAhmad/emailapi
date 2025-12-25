package webrisk

import (
	"context"
	"crypto/sha256"
	"fmt"

	webrisk "cloud.google.com/go/webrisk/apiv1"
	webriskpb "cloud.google.com/go/webrisk/apiv1/webriskpb"
	"google.golang.org/api/option"
)

const (
	// hashPrefixLength is the standard 4-byte prefix used by Web Risk.
	hashPrefixLength = 4
)

// client implements the Client interface using Google Cloud Web Risk API.
type client struct {
	webriskClient *webrisk.Client
	projectID     string
}

// NewClient creates a new Web Risk API client.
func NewClient(ctx context.Context, projectID, apiKey string) (Client, error) {
	opts := []option.ClientOption{
		option.WithAPIKey(apiKey),
	}

	c, err := webrisk.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create webrisk client: %w", err)
	}

	return &client{
		webriskClient: c,
		projectID:     projectID,
	}, nil
}

// Lookup verifies URLs against Web Risk Lookup API.
func (c *client) Lookup(ctx context.Context, urls []string) ([]ThreatMatch, error) {
	var matches []ThreatMatch

	// Web Risk API checks one URL at a time via SearchUris
	for _, url := range urls {
		resp, err := c.webriskClient.SearchUris(ctx, &webriskpb.SearchUrisRequest{
			Uri: url,
			ThreatTypes: []webriskpb.ThreatType{
				webriskpb.ThreatType_MALWARE,
				webriskpb.ThreatType_SOCIAL_ENGINEERING,
				webriskpb.ThreatType_UNWANTED_SOFTWARE,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search URI %s: %w", url, err)
		}

		if resp.Threat != nil && len(resp.Threat.ThreatTypes) > 0 {
			for _, tt := range resp.Threat.ThreatTypes {
				matches = append(matches, ThreatMatch{
					URL:        url,
					ThreatType: convertThreatType(tt),
				})
			}
		}
	}

	return matches, nil
}

// FetchHashPrefixes gets hash prefixes from Web Risk Update API.
func (c *client) FetchHashPrefixes(ctx context.Context, stateToken string) (*UpdateResponse, error) {
	threatTypes := []webriskpb.ThreatType{
		webriskpb.ThreatType_MALWARE,
		webriskpb.ThreatType_SOCIAL_ENGINEERING,
		webriskpb.ThreatType_UNWANTED_SOFTWARE,
	}

	var allPrefixes [][]byte

	// Fetch hash prefixes for each threat type
	for _, tt := range threatTypes {
		req := &webriskpb.ComputeThreatListDiffRequest{
			ThreatType: tt,
			Constraints: &webriskpb.ComputeThreatListDiffRequest_Constraints{
				MaxDiffEntries:     10000,
				MaxDatabaseEntries: 100000,
				SupportedCompressions: []webriskpb.CompressionType{
					webriskpb.CompressionType_RAW,
				},
			},
		}

		if stateToken != "" {
			req.VersionToken = []byte(stateToken)
		}

		resp, err := c.webriskClient.ComputeThreatListDiff(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to compute threat list diff for %v: %w", tt, err)
		}

		// Extract hash prefixes from additions
		if resp.Additions != nil && len(resp.Additions.RawHashes) > 0 {
			for _, rh := range resp.Additions.RawHashes {
				prefixSize := int(rh.PrefixSize)
				rawData := rh.RawHashes

				for i := 0; i+prefixSize <= len(rawData); i += prefixSize {
					prefix := make([]byte, prefixSize)
					copy(prefix, rawData[i:i+prefixSize])
					allPrefixes = append(allPrefixes, prefix)
				}
			}
		}
	}

	return &UpdateResponse{
		HashPrefixes:          allPrefixes,
		StateToken:            stateToken, // Would be from response in real impl
		NegativeCacheDuration: 300,        // 5 minutes default
	}, nil
}

// HashURL computes the SHA256 hash prefix for a URL.
func HashURL(url string) []byte {
	hash := sha256.Sum256([]byte(url))
	return hash[:hashPrefixLength]
}

// convertThreatType converts protobuf threat type to our type.
func convertThreatType(tt webriskpb.ThreatType) ThreatType {
	switch tt {
	case webriskpb.ThreatType_MALWARE:
		return ThreatTypeMalware
	case webriskpb.ThreatType_SOCIAL_ENGINEERING:
		return ThreatTypeSocialEngineering
	case webriskpb.ThreatType_UNWANTED_SOFTWARE:
		return ThreatTypeUnwantedSoftware
	default:
		return ThreatType(tt.String())
	}
}

// Close closes the underlying Web Risk client.
func (c *client) Close() error {
	return c.webriskClient.Close()
}
