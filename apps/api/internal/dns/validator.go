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
	"github.com/rs/zerolog/log"
)

const (
	// DefaultTimeout is the default timeout for a single DNS query.
	DefaultTimeout = 3 * time.Second

	// DefaultRetries is the number of retry attempts per resolver.
	DefaultRetries = 2

	// DefaultRetryDelay is the base delay for exponential backoff.
	DefaultRetryDelay = 100 * time.Millisecond

	// ConsensusThreshold is the minimum fraction of resolvers that must agree.
	// 0.5 means majority (at least half) must agree.
	ConsensusThreshold = 0.5
)

var (
	// DefaultResolvers are the DNS servers to query for consensus.
	// Using multiple providers ensures we don't rely on a single source.
	DefaultResolvers = []string{
		"8.8.8.8:53",        // Google Primary
		"8.8.4.4:53",        // Google Secondary
		"1.1.1.1:53",        // Cloudflare Primary
		"1.0.0.1:53",        // Cloudflare Secondary
		"9.9.9.9:53",        // Quad9
		"208.67.222.222:53", // OpenDNS
	}
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

// Key returns a unique identifier for the record (Type:Name).
// This is used for robust result mapping that doesn't depend on array ordering.
func (r ExpectedRecord) Key() string {
	return r.Type + ":" + r.Name
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
	Records   map[string]RecordResult // Keyed by ExpectedRecord.Key()
	CheckedAt time.Time
}

// ValidatorConfig holds configuration for the DNS validator.
type ValidatorConfig struct {
	Resolvers          []string
	Timeout            time.Duration
	Retries            int
	RetryDelay         time.Duration
	ConsensusThreshold float64
}

// DefaultConfig returns the default validator configuration.
func DefaultConfig() ValidatorConfig {
	return ValidatorConfig{
		Resolvers:          DefaultResolvers,
		Timeout:            DefaultTimeout,
		Retries:            DefaultRetries,
		RetryDelay:         DefaultRetryDelay,
		ConsensusThreshold: ConsensusThreshold,
	}
}

// Validator handles DNS record validation using DoH (DNS over HTTPS).
// DoH is more reliable than raw UDP queries as it uses HTTP/HTTPS.
type Validator struct {
	config    ValidatorConfig
	dohClient *DoHClient
}

// Ensure Validator implements ValidatorInterface
var _ ValidatorInterface = (*Validator)(nil)

// NewValidator creates a new DNS validator with DoH-based lookups.
func NewValidator() *Validator {
	return NewValidatorWithConfig(DefaultConfig())
}

// NewValidatorWithConfig creates a new DNS validator with custom configuration.
func NewValidatorWithConfig(config ValidatorConfig) *Validator {
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}
	if config.ConsensusThreshold == 0 {
		config.ConsensusThreshold = ConsensusThreshold
	}
	return &Validator{
		config:    config,
		dohClient: NewDoHClient(),
	}
}

// ValidateRecords checks multiple DNS records in parallel.
// Results are keyed by ExpectedRecord.Key() for order-independent mapping.
func (v *Validator) ValidateRecords(ctx context.Context, expected []ExpectedRecord) *ValidationResult {
	result := &ValidationResult{
		Records:   make(map[string]RecordResult, len(expected)),
		CheckedAt: time.Now(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, rec := range expected {
		wg.Add(1)
		go func(exp ExpectedRecord) {
			defer wg.Done()
			res := v.checkRecordWithConsensus(ctx, exp)
			mu.Lock()
			result.Records[exp.Key()] = res
			mu.Unlock()
		}(rec)
	}
	wg.Wait()

	return result
}

// checkRecordWithConsensus verifies a DNS record using DoH providers.
// Uses a simplified consensus: if ANY provider finds the record with correct value → FOUND.
func (v *Validator) checkRecordWithConsensus(ctx context.Context, expected ExpectedRecord) RecordResult {
	result := RecordResult{
		ExpectedRecord: expected,
		Status:         RecordStatusPending,
	}

	// Query DoH providers in parallel
	recordType := RecordTypeFromString(expected.Type)
	dohResults := v.dohClient.QueryAll(ctx, expected.Name, recordType)

	// Analyze results
	var foundMatching, foundMismatch, notFound, errors int
	var discoveredValues []string

	for _, r := range dohResults {
		if r.Error != nil {
			errors++
			continue
		}

		if !r.Found || len(r.Values) == 0 {
			notFound++
			continue
		}

		// Check if any value matches expected
		valueMatched := false
		for _, val := range r.Values {
			discoveredValues = append(discoveredValues, val)
			if v.valuesMatch(expected.Type, expected.Value, val) {
				valueMatched = true
			}
		}

		if valueMatched {
			foundMatching++
		} else {
			foundMismatch++
		}
	}

	// Use the most common discovered value
	if len(discoveredValues) > 0 {
		result.DiscoveredValue = getMostCommon(discoveredValues)
	}

	totalProviders := len(dohResults)
	successfulResponses := foundMatching + foundMismatch + notFound

	log.Debug().
		Str("record", expected.Name).
		Str("type", expected.Type).
		Int("foundMatching", foundMatching).
		Int("foundMismatch", foundMismatch).
		Int("notFound", notFound).
		Int("errors", errors).
		Int("providers", totalProviders).
		Msg("DoH DNS check")

	// Simplified consensus logic for DoH:
	// - If ANY provider finds the record with correct value → FOUND
	// - If providers find record but wrong value → MISMATCH
	// - If ALL successful providers say not found → MISSING
	// - If all providers failed → PENDING (try again later)

	if successfulResponses == 0 {
		// All providers failed - report as pending, not missing
		result.Status = RecordStatusPending
		result.Error = "all DNS providers failed"
		return result
	}

	if foundMatching > 0 {
		// At least one provider found the record with correct value
		result.Status = RecordStatusFound
	} else if foundMismatch > 0 {
		// Providers found record but with wrong value
		result.Status = RecordStatusMismatch
	} else {
		// All successful providers report not found
		result.Status = RecordStatusMissing
	}

	return result
}

// Legacy UDP types - kept for test compatibility
type resolverResult struct {
	resolver string
	value    string
	found    bool
	err      error
}


// querySingleResolver performs a single DNS query to a specific resolver.
func (v *Validator) querySingleResolver(ctx context.Context, resolver string, expected ExpectedRecord) (string, bool, error) {
	// Create a context with timeout for this specific query
	queryCtx, cancel := context.WithTimeout(ctx, v.config.Timeout)
	defer cancel()

	switch strings.ToUpper(expected.Type) {
	case "CNAME":
		return v.lookupCNAME(queryCtx, resolver, expected.Name)
	case "TXT":
		values, err := v.lookupTXT(queryCtx, resolver, expected.Name)
		if err != nil {
			return "", false, err
		}
		if len(values) == 0 {
			return "", false, nil
		}
		return strings.Join(values, "; "), true, nil
	case "MX":
		mxRecords, err := v.lookupMX(queryCtx, resolver, expected.Name)
		if err != nil {
			return "", false, err
		}
		if len(mxRecords) == 0 {
			return "", false, nil
		}
		parts := make([]string, len(mxRecords))
		for i, mx := range mxRecords {
			parts[i] = fmt.Sprintf("%d %s", mx.Priority, mx.Host)
		}
		return strings.Join(parts, "; "), true, nil
	default:
		return "", false, fmt.Errorf("unsupported record type: %s", expected.Type)
	}
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

// lookupCNAME looks up a CNAME record from a specific resolver.
func (v *Validator) lookupCNAME(ctx context.Context, resolver, name string) (string, bool, error) {
	name = dns.Fqdn(name)
	msg := new(dns.Msg)
	msg.SetQuestion(name, dns.TypeCNAME)

	resp, err := v.query(ctx, resolver, msg)
	if err != nil {
		return "", false, err
	}

	for _, ans := range resp.Answer {
		if cname, ok := ans.(*dns.CNAME); ok {
			return strings.TrimSuffix(cname.Target, "."), true, nil
		}
	}

	return "", false, nil
}

// lookupTXT looks up TXT records from a specific resolver.
func (v *Validator) lookupTXT(ctx context.Context, resolver, name string) ([]string, error) {
	name = dns.Fqdn(name)
	msg := new(dns.Msg)
	msg.SetQuestion(name, dns.TypeTXT)

	resp, err := v.query(ctx, resolver, msg)
	if err != nil {
		return nil, err
	}

	var values []string
	for _, ans := range resp.Answer {
		if txt, ok := ans.(*dns.TXT); ok {
			values = append(values, strings.Join(txt.Txt, ""))
		}
	}

	return values, nil
}

// MXRecord represents an MX record.
type MXRecord struct {
	Host     string
	Priority int
}

// lookupMX looks up MX records from a specific resolver.
func (v *Validator) lookupMX(ctx context.Context, resolver, name string) ([]MXRecord, error) {
	name = dns.Fqdn(name)
	msg := new(dns.Msg)
	msg.SetQuestion(name, dns.TypeMX)

	resp, err := v.query(ctx, resolver, msg)
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

	return records, nil
}

// query sends a DNS query to a specific resolver.
func (v *Validator) query(ctx context.Context, resolver string, msg *dns.Msg) (*dns.Msg, error) {
	// Create UDP connection with timeout
	dialer := net.Dialer{Timeout: v.config.Timeout}
	conn, err := dialer.DialContext(ctx, "udp", resolver)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", resolver, err)
	}
	defer conn.Close()

	// Set deadline based on context
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("failed to set deadline: %w", err)
		}
	}

	dnsConn := &dns.Conn{Conn: conn}

	if err := dnsConn.WriteMsg(msg); err != nil {
		return nil, fmt.Errorf("failed to write DNS query: %w", err)
	}

	resp, err := dnsConn.ReadMsg()
	if err != nil {
		return nil, fmt.Errorf("failed to read DNS response: %w", err)
	}

	if resp.Rcode == dns.RcodeNameError {
		// NXDOMAIN - domain doesn't exist
		return resp, nil
	}

	if resp.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("DNS query failed with rcode: %d", resp.Rcode)
	}

	return resp, nil
}

// getMostCommon returns the most common string from a slice.
func getMostCommon(values []string) string {
	if len(values) == 0 {
		return ""
	}

	counts := make(map[string]int)
	for _, v := range values {
		counts[v]++
	}

	maxCount := 0
	mostCommon := values[0]
	for v, count := range counts {
		if count > maxCount {
			maxCount = count
			mostCommon = v
		}
	}

	return mostCommon
}
