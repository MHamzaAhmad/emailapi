package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"

	"github.com/emailapi/api/internal/external/webrisk"
	"github.com/emailapi/api/internal/repository/redis"
)

const (
	// bloomFilterKey is the Redis key for the Web Risk hash prefix Bloom filter.
	bloomFilterKey = "webrisk:bloom"
)

// BodyValidator validates email body content for unsafe URLs.
type BodyValidator struct {
	redis   *redis.Client
	webrisk webrisk.Client
}

// NewBodyValidator creates a new BodyValidator.
func NewBodyValidator(redisClient *redis.Client, webriskClient webrisk.Client) *BodyValidator {
	return &BodyValidator{
		redis:   redisClient,
		webrisk: webriskClient,
	}
}

// ValidateURLs extracts URLs from body/html and checks them for threats.
// Uses a two-tier approach: fast Bloom filter check, then Web Risk API verification.
func (v *BodyValidator) ValidateURLs(ctx context.Context, body, htmlContent string) error {
	// Extract all URLs from content
	urls := v.extractURLs(body, htmlContent)
	if len(urls) == 0 {
		return nil
	}

	// Deduplicate URLs
	uniqueURLs := deduplicate(urls)

	// Compute hash prefixes for Bloom filter check
	hashPrefixes := make([]string, len(uniqueURLs))
	for i, u := range uniqueURLs {
		hashPrefixes[i] = computeHashPrefix(u)
	}

	// Fast path: check Bloom filter
	matches, err := v.redis.BFMExists(ctx, bloomFilterKey, hashPrefixes...)
	if err != nil {
		// On Bloom filter error, fall back to checking all URLs with API
		// This is a safe fallback but slower
		return v.verifyWithAPI(ctx, uniqueURLs)
	}

	// Collect URLs that matched Bloom filter (potential threats)
	var suspiciousURLs []string
	for i, matched := range matches {
		if matched {
			suspiciousURLs = append(suspiciousURLs, uniqueURLs[i])
		}
	}

	// If no Bloom filter matches, URLs are safe
	if len(suspiciousURLs) == 0 {
		return nil
	}

	// Verify suspicious URLs with Web Risk API
	return v.verifyWithAPI(ctx, suspiciousURLs)
}

// verifyWithAPI checks URLs against Web Risk Lookup API.
func (v *BodyValidator) verifyWithAPI(ctx context.Context, urls []string) error {
	if v.webrisk == nil {
		return nil // No Web Risk client configured, skip verification
	}

	threats, err := v.webrisk.Lookup(ctx, urls)
	if err != nil {
		// Log but don't block on API errors
		return nil
	}

	if len(threats) == 0 {
		return nil
	}

	// Build validation errors for all threats found
	errors := NewValidationErrors()
	for _, threat := range threats {
		errors.Add(UnsafeURLError("body", threat.URL, string(threat.ThreatType)))
	}

	return errors
}

// extractURLs extracts URLs from both plain text and HTML content.
func (v *BodyValidator) extractURLs(body, htmlContent string) []string {
	var urls []string

	// Extract from plain text body
	if body != "" {
		urls = append(urls, extractURLsFromText(body)...)
	}

	// Extract from HTML
	if htmlContent != "" {
		urls = append(urls, extractURLsFromHTML(htmlContent)...)
	}

	return urls
}

// urlRegex matches common URL patterns.
var urlRegex = regexp.MustCompile(`https?://[^\s<>"']+`)

// extractURLsFromText extracts URLs from plain text.
func extractURLsFromText(text string) []string {
	matches := urlRegex.FindAllString(text, -1)
	var urls []string
	for _, m := range matches {
		// Clean trailing punctuation
		m = strings.TrimRight(m, ".,;:!?)")
		if u, err := url.Parse(m); err == nil && u.Host != "" {
			urls = append(urls, m)
		}
	}
	return urls
}

// extractURLsFromHTML extracts URLs from href, src, and other attributes.
func extractURLsFromHTML(htmlContent string) []string {
	var urls []string

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// Fall back to regex extraction on parse error
		return extractURLsFromText(htmlContent)
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Check href and src attributes
			for _, attr := range n.Attr {
				if attr.Key == "href" || attr.Key == "src" || attr.Key == "action" {
					if strings.HasPrefix(attr.Val, "http://") || strings.HasPrefix(attr.Val, "https://") {
						urls = append(urls, attr.Val)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)

	// Also extract URLs from text nodes (inline URLs in content)
	urls = append(urls, extractURLsFromText(htmlContent)...)

	return urls
}

// computeHashPrefix computes the 4-byte SHA256 hash prefix for a URL.
func computeHashPrefix(u string) string {
	hash := sha256.Sum256([]byte(u))
	return hex.EncodeToString(hash[:4])
}

// deduplicate removes duplicate URLs.
func deduplicate(urls []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, u := range urls {
		if !seen[u] {
			seen[u] = true
			result = append(result, u)
		}
	}
	return result
}

// UnsafeURLError creates an error for a URL flagged as unsafe.
func UnsafeURLError(field, urlVal, threatType string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Code:    ErrCodeUnsafeURL,
		Message: fmt.Sprintf("URL '%s' is flagged as %s and cannot be included in emails", urlVal, threatType),
	}
}
