// Package dns provides DNS validation for domain verification.
package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

const (
	// DefaultTimeout is the default timeout for DNS queries.
	DefaultTimeout = 5 * time.Second
)

var (
	// DefaultResolvers are the DNS servers to query.
	DefaultResolvers = []string{"8.8.8.8:53", "1.1.1.1:53"}
)

// RecordStatus represents the status of a DNS record check.
type RecordStatus string

const (
	RecordStatusPending  RecordStatus = "pending"
	RecordStatusFound    RecordStatus = "found"
	RecordStatusMismatch RecordStatus = "mismatch"
	RecordStatusMissing  RecordStatus = "missing"
)

// ExpectedRecord describes a DNS record we expect to find.
type ExpectedRecord struct {
	Type     string // CNAME, TXT, MX
	Name     string // Full DNS name
	Value    string // Expected value
	Priority int    // For MX records
}

// RecordResult contains the result of checking a single DNS record.
type RecordResult struct {
	ExpectedRecord
	Status          RecordStatus
	DiscoveredValue string
	Error           string
}

// ValidationResult contains results of DNS validation for a domain.
type ValidationResult struct {
	Records   []RecordResult
	CheckedAt time.Time
}

// Validator handles DNS record validation.
type Validator struct {
	resolvers []string
	timeout   time.Duration
	client    *dns.Client
}

// NewValidator creates a new DNS validator.
func NewValidator() *Validator {
	return &Validator{
		resolvers: DefaultResolvers,
		timeout:   DefaultTimeout,
		client: &dns.Client{
			Timeout: DefaultTimeout,
			Net:     "udp",
		},
	}
}

// ValidateRecords checks multiple DNS records in parallel.
func (v *Validator) ValidateRecords(ctx context.Context, expected []ExpectedRecord) *ValidationResult {
	result := &ValidationResult{
		Records:   make([]RecordResult, len(expected)),
		CheckedAt: time.Now(),
	}

	var wg sync.WaitGroup
	for i, rec := range expected {
		wg.Add(1)
		go func(idx int, exp ExpectedRecord) {
			defer wg.Done()
			result.Records[idx] = v.checkRecord(ctx, exp)
		}(i, rec)
	}
	wg.Wait()

	return result
}

// checkRecord verifies a single DNS record.
func (v *Validator) checkRecord(ctx context.Context, expected ExpectedRecord) RecordResult {
	result := RecordResult{
		ExpectedRecord: expected,
		Status:         RecordStatusMissing,
	}

	var discoveredValue string
	var err error

	switch strings.ToUpper(expected.Type) {
	case "CNAME":
		discoveredValue, err = v.lookupCNAME(ctx, expected.Name)
	case "TXT":
		var values []string
		values, err = v.lookupTXT(ctx, expected.Name)
		if len(values) > 0 {
			discoveredValue = strings.Join(values, "; ")
		}
	case "MX":
		var mxRecords []MXRecord
		mxRecords, err = v.lookupMX(ctx, expected.Name)
		if len(mxRecords) > 0 {
			// Format: "10 mail.example.com"
			parts := make([]string, len(mxRecords))
			for i, mx := range mxRecords {
				parts[i] = fmt.Sprintf("%d %s", mx.Priority, mx.Host)
			}
			discoveredValue = strings.Join(parts, "; ")
		}
	default:
		result.Error = fmt.Sprintf("unsupported record type: %s", expected.Type)
		return result
	}

	result.DiscoveredValue = discoveredValue

	if err != nil {
		// Check if it's a NXDOMAIN (record doesn't exist)
		if strings.Contains(err.Error(), "NXDOMAIN") || strings.Contains(err.Error(), "no such host") {
			result.Status = RecordStatusMissing
		} else {
			result.Error = err.Error()
			result.Status = RecordStatusMissing
		}
		return result
	}

	if discoveredValue == "" {
		result.Status = RecordStatusMissing
		return result
	}

	// Check if value matches
	if v.valuesMatch(expected.Type, expected.Value, discoveredValue) {
		result.Status = RecordStatusFound
	} else {
		result.Status = RecordStatusMismatch
	}

	return result
}

// valuesMatch compares expected and discovered values with type-specific normalization.
func (v *Validator) valuesMatch(recordType, expected, discovered string) bool {
	// Normalize values
	expected = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(expected), "."))
	discovered = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(discovered), "."))

	switch strings.ToUpper(recordType) {
	case "CNAME":
		return expected == discovered
	case "TXT":
		// TXT records may have multiple values, check if expected is contained
		return strings.Contains(discovered, expected)
	case "MX":
		// Check if the MX host matches (ignore priority)
		return strings.Contains(discovered, expected)
	default:
		return expected == discovered
	}
}

// lookupCNAME looks up a CNAME record.
func (v *Validator) lookupCNAME(ctx context.Context, name string) (string, error) {
	name = dns.Fqdn(name)
	msg := new(dns.Msg)
	msg.SetQuestion(name, dns.TypeCNAME)

	resp, err := v.query(ctx, msg)
	if err != nil {
		return "", err
	}

	for _, ans := range resp.Answer {
		if cname, ok := ans.(*dns.CNAME); ok {
			return strings.TrimSuffix(cname.Target, "."), nil
		}
	}

	return "", fmt.Errorf("no CNAME record found for %s", name)
}

// lookupTXT looks up TXT records.
func (v *Validator) lookupTXT(ctx context.Context, name string) ([]string, error) {
	name = dns.Fqdn(name)
	msg := new(dns.Msg)
	msg.SetQuestion(name, dns.TypeTXT)

	resp, err := v.query(ctx, msg)
	if err != nil {
		return nil, err
	}

	var values []string
	for _, ans := range resp.Answer {
		if txt, ok := ans.(*dns.TXT); ok {
			values = append(values, strings.Join(txt.Txt, ""))
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no TXT record found for %s", name)
	}

	return values, nil
}

// MXRecord represents an MX record.
type MXRecord struct {
	Host     string
	Priority int
}

// lookupMX looks up MX records.
func (v *Validator) lookupMX(ctx context.Context, name string) ([]MXRecord, error) {
	name = dns.Fqdn(name)
	msg := new(dns.Msg)
	msg.SetQuestion(name, dns.TypeMX)

	resp, err := v.query(ctx, msg)
	if err != nil {
		return nil, err
	}

	var records []MXRecord
	for _, ans := range resp.Answer {
		if mx, ok := ans.(*dns.MX); ok {
			records = append(records, MXRecord{
				Host:     strings.TrimSuffix(mx.Mx, "."),
				Priority: int(mx.Preference),
			})
		}
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no MX record found for %s", name)
	}

	return records, nil
}

// query sends a DNS query to resolvers and returns the first successful response.
func (v *Validator) query(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	var lastErr error

	for _, resolver := range v.resolvers {
		// Create a new connection for each resolver
		conn, err := net.DialTimeout("udp", resolver, v.timeout)
		if err != nil {
			lastErr = err
			continue
		}
		defer conn.Close()

		dnsConn := &dns.Conn{Conn: conn}
		defer dnsConn.Close()

		if err := dnsConn.WriteMsg(msg); err != nil {
			lastErr = err
			continue
		}

		resp, err := dnsConn.ReadMsg()
		if err != nil {
			lastErr = err
			continue
		}

		if resp.Rcode == dns.RcodeNameError {
			return nil, fmt.Errorf("NXDOMAIN: domain does not exist")
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all DNS resolvers failed: %w", lastErr)
	}

	return nil, fmt.Errorf("no DNS resolvers configured")
}
