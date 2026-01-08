// Package dns provides DNS validation for domain verification.
package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// DoH (DNS over HTTPS) providers for reliable DNS lookups.
var DefaultDoHProviders = []string{
	"https://dns.google/resolve",
	"https://cloudflare-dns.com/dns-query",
}

// DNS record type codes.
const (
	TypeA     = 1
	TypeCNAME = 5
	TypeMX    = 15
	TypeTXT   = 16
)

// DoHTimeout is the timeout for DoH HTTP requests.
const DoHTimeout = 10 * time.Second

// DoHResponse represents the JSON response from DoH providers.
type DoHResponse struct {
	Status   int           `json:"Status"` // 0 = NOERROR, 3 = NXDOMAIN
	TC       bool          `json:"TC"`     // Truncated
	RD       bool          `json:"RD"`     // Recursion desired
	RA       bool          `json:"RA"`     // Recursion available
	AD       bool          `json:"AD"`     // Authenticated data (DNSSEC)
	CD       bool          `json:"CD"`     // Checking disabled
	Question []DoHQuestion `json:"Question"`
	Answer   []DoHAnswer   `json:"Answer"`
}

// DoHQuestion represents a question in the DoH response.
type DoHQuestion struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

// DoHAnswer represents an answer record in the DoH response.
type DoHAnswer struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}

// DoHClient performs DNS lookups via DNS over HTTPS.
type DoHClient struct {
	httpClient *http.Client
	providers  []string
}

// NewDoHClient creates a new DoH client with default providers.
func NewDoHClient() *DoHClient {
	return NewDoHClientWithProviders(DefaultDoHProviders)
}

// NewDoHClientWithProviders creates a DoH client with custom providers.
func NewDoHClientWithProviders(providers []string) *DoHClient {
	if len(providers) == 0 {
		providers = DefaultDoHProviders
	}
	return &DoHClient{
		httpClient: &http.Client{
			Timeout: DoHTimeout,
		},
		providers: providers,
	}
}

// DoHResult contains the result from a single DoH provider.
type DoHResult struct {
	Provider string
	Values   []string
	Found    bool
	Error    error
}

// QueryAll queries all providers in parallel and returns all results.
func (c *DoHClient) QueryAll(ctx context.Context, name string, recordType int) []DoHResult {
	results := make([]DoHResult, len(c.providers))
	var wg sync.WaitGroup

	for i, provider := range c.providers {
		wg.Add(1)
		go func(idx int, prov string) {
			defer wg.Done()
			results[idx] = c.query(ctx, prov, name, recordType)
		}(i, provider)
	}

	wg.Wait()
	return results
}

// query performs a single DoH query to a provider.
func (c *DoHClient) query(ctx context.Context, provider, name string, recordType int) DoHResult {
	result := DoHResult{Provider: provider}

	// Construct URL with query parameters
	u, err := url.Parse(provider)
	if err != nil {
		result.Error = fmt.Errorf("invalid provider URL: %w", err)
		return result
	}

	q := u.Query()
	q.Set("name", name)
	q.Set("type", fmt.Sprintf("%d", recordType))
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		return result
	}

	// Set Accept header for JSON response
	req.Header.Set("Accept", "application/dns-json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("HTTP request failed: %w", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("HTTP status %d", resp.StatusCode)
		return result
	}

	// Parse JSON response
	var dohResp DoHResponse
	if err := json.NewDecoder(resp.Body).Decode(&dohResp); err != nil {
		result.Error = fmt.Errorf("failed to parse response: %w", err)
		return result
	}

	// Check DNS status
	if dohResp.Status == 3 {
		// NXDOMAIN - domain doesn't exist
		result.Found = false
		return result
	}

	if dohResp.Status != 0 {
		result.Error = fmt.Errorf("DNS error status: %d", dohResp.Status)
		return result
	}

	// Extract values from answers
	for _, ans := range dohResp.Answer {
		if ans.Type == recordType {
			result.Values = append(result.Values, normalizeRecordValue(ans.Data))
			result.Found = true
		}
	}

	log.Debug().
		Str("provider", provider).
		Str("name", name).
		Int("type", recordType).
		Bool("found", result.Found).
		Int("values", len(result.Values)).
		Msg("DoH query result")

	return result
}

// normalizeRecordValue cleans up a DNS record value.
func normalizeRecordValue(value string) string {
	// Remove trailing dot from FQDNs
	value = strings.TrimSuffix(value, ".")
	// Remove quotes from TXT records
	value = strings.Trim(value, "\"")
	return value
}

// RecordTypeFromString converts a record type string to its numeric code.
func RecordTypeFromString(t string) int {
	switch strings.ToUpper(t) {
	case "A":
		return TypeA
	case "CNAME":
		return TypeCNAME
	case "MX":
		return TypeMX
	case "TXT":
		return TypeTXT
	default:
		return TypeA
	}
}
