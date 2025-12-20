package domain

import (
	"time"
)

// DomainStatus represents the verification status of a domain.
type DomainStatus string

const (
	DomainStatusPending          DomainStatus = "pending"
	DomainStatusSuccess          DomainStatus = "success"
	DomainStatusFailed           DomainStatus = "failed"
	DomainStatusTemporaryFailure DomainStatus = "temporary_failure"
)

// RecordStatus represents the verification status of an individual DNS record.
type RecordStatus string

const (
	RecordStatusPending  RecordStatus = "pending"
	RecordStatusVerified RecordStatus = "verified"
	RecordStatusFailed   RecordStatus = "failed"
)

// RecordType identifies the type and purpose of a DNS record.
type RecordType string

const (
	RecordTypeDKIM        RecordType = "dkim"
	RecordTypeSPF         RecordType = "spf"
	RecordTypeDMARC       RecordType = "dmarc"
	RecordTypeMXInbound   RecordType = "mx_inbound"
	RecordTypeMailFromMX  RecordType = "mail_from_mx"
	RecordTypeMailFromSPF RecordType = "mail_from_spf"
)

// SendingDomain represents a verified sending domain in the system.
type SendingDomain struct {
	ID                 string       `json:"id"`
	UserID             string       `json:"user_id"`
	Domain             string       `json:"domain"`
	Status             DomainStatus `json:"status"`
	VerifiedForSending bool         `json:"verified_for_sending"`

	// DKIM configuration
	DkimTokens []string     `json:"dkim_tokens,omitempty"`
	DkimStatus DomainStatus `json:"dkim_status"`

	// Custom MAIL FROM configuration
	MailFromDomain string       `json:"mail_from_domain,omitempty"`
	MailFromStatus DomainStatus `json:"mail_from_status,omitempty"`

	// AWS region where the domain is registered
	Region string `json:"region"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DnsRecord represents a single DNS record that needs to be configured.
type DnsRecord struct {
	// DNS record type (e.g., CNAME, TXT, MX)
	DnsType string `json:"dns_type"`

	// Record name/host (e.g., "selector1._domainkey.example.com")
	Name string `json:"name"`

	// Record value to set
	Value string `json:"value"`

	// Priority for MX records (ignored for other types)
	Priority int `json:"priority,omitempty"`

	// Purpose of this record
	RecordType RecordType `json:"record_type"`

	// Current verification status
	Status RecordStatus `json:"status"`

	// Human-readable instructions
	Instructions string `json:"instructions,omitempty"`
}

// DomainRecords contains all DNS records needed for email deliverability.
type DomainRecords struct {
	Domain string `json:"domain"`

	// DKIM records (3 CNAME records for email signing)
	DkimRecords []DnsRecord `json:"dkim_records"`

	// SPF record (TXT record authorizing SES)
	SpfRecord *DnsRecord `json:"spf_record,omitempty"`

	// DMARC record (TXT record for authentication policy)
	DmarcRecord *DnsRecord `json:"dmarc_record,omitempty"`

	// MX records for inbound email
	MxRecords []DnsRecord `json:"mx_records,omitempty"`

	// MAIL FROM records (MX + SPF for custom return path)
	MailFromRecords []DnsRecord `json:"mail_from_records,omitempty"`

	// Status flags
	IsReadyToSend     bool `json:"is_ready_to_send"`
	IsReadyToReceive  bool `json:"is_ready_to_receive"`
	IsFullyConfigured bool `json:"is_fully_configured"`
}

// AddDomainRequest represents a request to add a new domain.
type AddDomainRequest struct {
	Domain string `json:"domain" binding:"required"`
}

// SetMailFromRequest represents a request to configure custom MAIL FROM.
type SetMailFromRequest struct {
	MailFromSubdomain string `json:"mail_from_subdomain" binding:"required"`
}
