package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// MaxDownloadSize is the maximum size of a file that can be downloaded from a URL (25MB)
	MaxDownloadSize = 25 * 1024 * 1024
	// DownloadTimeout is the timeout for downloading from a URL
	DownloadTimeout = 30 * time.Second
)

// DownloadFromURL downloads content from an external URL with size and timeout limits.
// Returns the content, content-type, and any error.
func DownloadFromURL(ctx context.Context, url string) ([]byte, string, error) {
	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Create client with timeout
	client := &http.Client{
		Timeout: DownloadTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check content length if provided
	if resp.ContentLength > MaxDownloadSize {
		return nil, "", fmt.Errorf("file too large: %d bytes (max %d)", resp.ContentLength, MaxDownloadSize)
	}

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, MaxDownloadSize+1)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	if len(content) > MaxDownloadSize {
		return nil, "", fmt.Errorf("file too large: exceeds %d bytes", MaxDownloadSize)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return content, contentType, nil
}
