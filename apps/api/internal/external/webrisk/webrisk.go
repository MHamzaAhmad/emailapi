package webrisk

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
