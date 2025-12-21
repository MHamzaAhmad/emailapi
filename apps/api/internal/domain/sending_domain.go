package domain

import (
	"time"
)

// DomainStatus represents the overall status of a domain.
// Designed for easy comparison: if domain.Status == DomainStatusReady
type DomainStatus string

const (
	DomainStatusPending   DomainStatus = "pending"   // Just added, waiting for DNS
	DomainStatusVerifying DomainStatus = "verifying" // DNS check in progress
	DomainStatusReady     DomainStatus = "ready"     // Can send emails!
	DomainStatusDegraded  DomainStatus = "degraded"  // Works but missing optional records
	DomainStatusFailed    DomainStatus = "failed"    // Cannot send, action required
)

// RecordStatus represents the status of a single DNS record.
// Designed for easy comparison: if record.Status == RecordStatusFound
type RecordStatus string

const (
	RecordStatusPending  RecordStatus = "pending"  // Not checked yet
	RecordStatusFound    RecordStatus = "found"    // Found with correct value ✓
	RecordStatusMismatch RecordStatus = "mismatch" // Found but wrong value
	RecordStatusMissing  RecordStatus = "missing"  // Not found in DNS
)

// RecordType identifies the purpose of a DNS record.
type RecordType string

const (
	RecordTypeDKIM        RecordType = "dkim"
	RecordTypeSPF         RecordType = "spf"
	RecordTypeDMARC       RecordType = "dmarc"
	RecordTypeMXInbound   RecordType = "mx_inbound"
	RecordTypeMailFromMX  RecordType = "mail_from_mx"
	RecordTypeMailFromSPF RecordType = "mail_from_spf"
)

// SendingDomain represents a sending domain with all its configuration.
type SendingDomain struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	Domain    string       `json:"domain"`
	Status    DomainStatus `json:"status"`
	Region    string       `json:"region"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`

	// LastCheckedAt tracks when DNS/SES was last checked.
	// Used for rate limiting and stale data refresh.
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`

	// DKIM configuration (stored for building records)
	DkimTokens []string     `json:"dkim_tokens,omitempty"`
	DkimStatus DomainStatus `json:"dkim_status"`

	// MAIL FROM configuration (auto-configured on add)
	MailFromDomain string       `json:"mail_from_domain,omitempty"`
	MailFromStatus DomainStatus `json:"mail_from_status,omitempty"`

	// SES verification status
	VerifiedForSending bool `json:"verified_for_sending"`
}

// DomainSummary provides human-readable status for lazy users.
type DomainSummary struct {
	Message           string `json:"message"`
	NextAction        string `json:"next_action"` // CONFIGURE_DNS, WAIT, NONE
	RecordsPending    int    `json:"records_pending"`
	RecordsConfigured int    `json:"records_configured"`
	CanSend           bool   `json:"can_send"`
	CanReceive        bool   `json:"can_receive"`
}

// DnsRecord represents a single DNS record to configure.
type DnsRecord struct {
	Type            string       `json:"type"`     // CNAME, TXT, MX
	Name            string       `json:"name"`     // Full name
	Value           string       `json:"value"`    // Expected value
	Priority        int          `json:"priority"` // For MX records
	RecordType      RecordType   `json:"record_type"`
	Status          RecordStatus `json:"status"`
	NameShort       string       `json:"name_short"`       // For DNS providers wanting just subdomain
	DiscoveredValue string       `json:"discovered_value"` // What we found in DNS
	Instructions    string       `json:"instructions"`
}

// DomainRecords contains all DNS records needed for email.
type DomainRecords struct {
	DkimRecords     []DnsRecord `json:"dkim_records"`
	SpfRecord       *DnsRecord  `json:"spf_record,omitempty"`
	DmarcRecord     *DnsRecord  `json:"dmarc_record,omitempty"`
	MxRecords       []DnsRecord `json:"mx_records,omitempty"`
	MailFromRecords []DnsRecord `json:"mail_from_records,omitempty"`
}

// DomainWithDetails combines domain info with records and summary.
type DomainWithDetails struct {
	*SendingDomain
	Summary *DomainSummary `json:"summary"`
	Records *DomainRecords `json:"records"`
}

// VerifyResult contains the outcome of a verification attempt.
type VerifyResult struct {
	Domain       *DomainWithDetails `json:"domain"`
	WasRefreshed bool               `json:"was_refreshed"`
	NextRetryAt  *time.Time         `json:"next_retry_at,omitempty"`
	Message      string             `json:"message,omitempty"`
}
