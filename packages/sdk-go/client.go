// Package emailapi provides a Go SDK for the Email API.
package emailapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.emailapi.dev/v1"
	defaultTimeout = 30 * time.Second
)

// Client is the Email API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client

	// API namespaces
	Emails   *EmailsAPI
	Webhooks *WebhooksAPI
	Users    *UsersAPI
}

// ClientOption is a function that configures the client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new Email API client.
//
// Example:
//
//	client := emailapi.NewClient("em_...")
//
//	resp, err := client.Emails.Send(ctx, &emailapi.SendEmailRequest{
//		From:    "sender@example.com",
//		To:      []string{"recipient@example.com"},
//		Subject: "Hello",
//		Body:    "World",
//	})
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize API namespaces
	c.Emails = &EmailsAPI{client: c}
	c.Webhooks = &WebhooksAPI{client: c}
	c.Users = &UsersAPI{client: c}

	return c
}

// request makes an authenticated request to the API.
func (c *Client) request(ctx context.Context, method, path string, body, result interface{}) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}
		apiErr.StatusCode = resp.StatusCode
		return &apiErr
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// APIError represents an API error response.
type APIError struct {
	Message    string `json:"error"`
	Details    string `json:"details,omitempty"`
	StatusCode int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

// EmailsAPI provides email operations.
type EmailsAPI struct {
	client *Client
}

// Send sends an email.
func (a *EmailsAPI) Send(ctx context.Context, req *SendEmailRequest) (*SendEmailResponse, error) {
	var resp SendEmailResponse
	if err := a.client.request(ctx, "POST", "/send", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get retrieves an email by ID.
func (a *EmailsAPI) Get(ctx context.Context, id string) (*Email, error) {
	var email Email
	if err := a.client.request(ctx, "GET", "/emails/"+id, nil, &email); err != nil {
		return nil, err
	}
	return &email, nil
}

// List retrieves a list of emails.
func (a *EmailsAPI) List(ctx context.Context, opts *ListOptions) (*EmailListResponse, error) {
	path := "/emails"
	if opts != nil {
		path += fmt.Sprintf("?limit=%d&offset=%d", opts.Limit, opts.Offset)
	}
	var resp EmailListResponse
	if err := a.client.request(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// WebhooksAPI provides webhook operations.
type WebhooksAPI struct {
	client *Client
}

// Create creates a new webhook.
func (a *WebhooksAPI) Create(ctx context.Context, req *CreateWebhookRequest) (*WebhookWithSecret, error) {
	var resp WebhookWithSecret
	if err := a.client.request(ctx, "POST", "/webhooks", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get retrieves a webhook by ID.
func (a *WebhooksAPI) Get(ctx context.Context, id string) (*Webhook, error) {
	var webhook Webhook
	if err := a.client.request(ctx, "GET", "/webhooks/"+id, nil, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

// List retrieves all webhooks.
func (a *WebhooksAPI) List(ctx context.Context) (*WebhookListResponse, error) {
	var resp WebhookListResponse
	if err := a.client.request(ctx, "GET", "/webhooks", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Update updates a webhook.
func (a *WebhooksAPI) Update(ctx context.Context, id string, req *UpdateWebhookRequest) (*Webhook, error) {
	var webhook Webhook
	if err := a.client.request(ctx, "PATCH", "/webhooks/"+id, req, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

// Delete deletes a webhook.
func (a *WebhooksAPI) Delete(ctx context.Context, id string) error {
	return a.client.request(ctx, "DELETE", "/webhooks/"+id, nil, nil)
}

// UsersAPI provides user operations.
type UsersAPI struct {
	client *Client
}

// Me retrieves the current user.
func (a *UsersAPI) Me(ctx context.Context) (*User, error) {
	var user User
	if err := a.client.request(ctx, "GET", "/users/me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// RegenerateAPIKey generates a new API key.
func (a *UsersAPI) RegenerateAPIKey(ctx context.Context) (*APIKeyResponse, error) {
	var resp APIKeyResponse
	if err := a.client.request(ctx, "POST", "/users/me/api-key", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListOptions specifies pagination options.
type ListOptions struct {
	Limit  int
	Offset int
}
