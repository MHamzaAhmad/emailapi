package tinybird

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is an HTTP client for interacting with the Tinybird API.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Tinybird API client.
func NewClient(token, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.tinybird.co"
	}
	return &Client{
		token:   token,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EventsPayload represents a batch of events to send to a datasource.
type EventsPayload struct {
	Events []map[string]interface{}
}

// IngestEvent sends a single event to a Tinybird datasource via the Events API.
func (c *Client) IngestEvent(ctx context.Context, datasource string, event map[string]interface{}) error {
	return c.IngestEvents(ctx, datasource, []map[string]interface{}{event})
}

// IngestEvents sends multiple events to a Tinybird datasource via the Events API.
// Uses NDJSON format for efficient batch ingestion.
func (c *Client) IngestEvents(ctx context.Context, datasource string, events []map[string]interface{}) error {
	if len(events) == 0 {
		return nil
	}

	// Build NDJSON payload
	var buf bytes.Buffer
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	endpoint := fmt.Sprintf("%s/v0/events?name=%s", c.baseURL, url.QueryEscape(datasource))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tinybird events API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// PipeResponse represents the response from a Tinybird pipe endpoint.
type PipeResponse struct {
	Data     []map[string]interface{} `json:"data"`
	Meta     []PipeMeta               `json:"meta"`
	Rows     int                      `json:"rows"`
	RowsRead int                      `json:"rows_read"`
}

// PipeMeta represents column metadata in a pipe response.
type PipeMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// QueryPipe queries a Tinybird pipe endpoint with the given parameters.
func (c *Client) QueryPipe(ctx context.Context, pipeName string, params map[string]string) (*PipeResponse, error) {
	endpoint := fmt.Sprintf("%s/v0/pipes/%s.json", c.baseURL, url.PathEscape(pipeName))

	// Build query string
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Set(k, v)
		}
		endpoint += "?" + values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query pipe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tinybird pipe query error (status %d): %s", resp.StatusCode, string(body))
	}

	var pipeResp PipeResponse
	if err := json.NewDecoder(resp.Body).Decode(&pipeResp); err != nil {
		return nil, fmt.Errorf("failed to decode pipe response: %w", err)
	}

	return &pipeResp, nil
}

// Ping checks if the Tinybird API is reachable and the token is valid.
func (c *Client) Ping(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/v0/datasources", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to ping tinybird: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return fmt.Errorf("tinybird ping failed with status %d", resp.StatusCode)
	}

	return nil
}
