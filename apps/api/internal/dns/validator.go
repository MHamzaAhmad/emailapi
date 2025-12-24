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

// Validator handles DNS record validation with consensus-based lookups.
type Validator struct {
	config ValidatorConfig
}

// NewValidator creates a new DNS validator with default configuration.
func NewValidator() *Validator {
	return NewValidatorWithConfig(DefaultConfig())
}

// NewValidatorWithConfig creates a new DNS validator with custom configuration.
func NewValidatorWithConfig(config ValidatorConfig) *Validator {
	if len(config.Resolvers) == 0 {
		config.Resolvers = DefaultResolvers
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}
	if config.ConsensusThreshold == 0 {
		config.ConsensusThreshold = ConsensusThreshold
	}
	return &Validator{config: config}
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
			result.Records[idx] = v.checkRecordWithConsensus(ctx, exp)
		}(i, rec)
	}
	wg.Wait()

	return result
}

// resolverResult holds the result from a single resolver query.
type resolverResult struct {
	resolver string
	value    string
	found    bool
	err      error
}

// checkRecordWithConsensus verifies a DNS record using multiple resolvers.
// It requires a majority of resolvers to agree for a "found" status.
func (v *Validator) checkRecordWithConsensus(ctx context.Context, expected ExpectedRecord) RecordResult {
	result := RecordResult{
		ExpectedRecord: expected,
		Status:         RecordStatusMissing,
	}

	// Query all resolvers in parallel
	results := v.queryAllResolvers(ctx, expected)

	// Analyze results for consensus
	foundCount := 0
	mismatchCount := 0
	missingCount := 0
	var discoveredValues []string

	for _, r := range results {
		if r.err != nil {
			// Treat errors as missing (resolver failure)
			missingCount++
			continue
		}

		if !r.found || r.value == "" {
			missingCount++
			continue
		}

		// Record was found, check if value matches
		if v.valuesMatch(expected.Type, expected.Value, r.value) {
			foundCount++
			discoveredValues = append(discoveredValues, r.value)
		} else {
			mismatchCount++
			discoveredValues = append(discoveredValues, r.value)
		}
	}

	// Use the most common discovered value
	if len(discoveredValues) > 0 {
		result.DiscoveredValue = getMostCommon(discoveredValues)
	}

	// Calculate consensus
	totalResponses := foundCount + mismatchCount + missingCount
	if totalResponses == 0 {
		result.Status = RecordStatusMissing
		result.Error = "all DNS resolvers failed"
		return result
	}

	threshold := int(float64(len(v.config.Resolvers)) * v.config.ConsensusThreshold)
	if threshold < 1 {
		threshold = 1
	}

	log.Debug().
		Str("record", expected.Name).
		Str("type", expected.Type).
		Int("found", foundCount).
		Int("mismatch", mismatchCount).
		Int("missing", missingCount).
		Int("threshold", threshold).
		Msg("DNS consensus check")

	// Determine status based on consensus
	if foundCount >= threshold {
		result.Status = RecordStatusFound
	} else if mismatchCount >= threshold {
		result.Status = RecordStatusMismatch
	} else if missingCount >= threshold {
		result.Status = RecordStatusMissing
	} else {
		// No clear consensus - be conservative and report missing
		// This handles cases where results are split
		result.Status = RecordStatusMissing
		result.Error = "no consensus among DNS resolvers"
	}

	return result
}

// queryAllResolvers queries all configured resolvers in parallel with retries.
func (v *Validator) queryAllResolvers(ctx context.Context, expected ExpectedRecord) []resolverResult {
	results := make([]resolverResult, len(v.config.Resolvers))
	var wg sync.WaitGroup

	for i, resolver := range v.config.Resolvers {
		wg.Add(1)
		go func(idx int, resolver string) {
			defer wg.Done()
			results[idx] = v.queryWithRetry(ctx, resolver, expected)
		}(i, resolver)
	}

	wg.Wait()
	return results
}

// queryWithRetry queries a single resolver with exponential backoff retry.
func (v *Validator) queryWithRetry(ctx context.Context, resolver string, expected ExpectedRecord) resolverResult {
	result := resolverResult{resolver: resolver}

	for attempt := 0; attempt <= v.config.Retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms, ...
			delay := v.config.RetryDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				result.err = ctx.Err()
				return result
			case <-time.After(delay):
			}
		}

		value, found, err := v.querySingleResolver(ctx, resolver, expected)
		if err == nil {
			result.value = value
			result.found = found
			return result
		}

		result.err = err
		// Continue to retry on error
	}

	return result
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
