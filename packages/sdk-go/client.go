// Package emailapi provides a Go SDK for the Email API using native gRPC/Connect.
package emailapi

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/emailapi/sdk-go/gen/v1/emailapiv1connect"
)

const (
	defaultBaseURL = "https://api.emailapi.dev"
)

// Client is the Email API client.
type Client struct {
	Emails   emailapiv1connect.EmailServiceClient
	Domains  emailapiv1connect.DomainServiceClient
	Users    emailapiv1connect.UserServiceClient
	Webhooks emailapiv1connect.WebhookServiceClient
}

// ClientOption configures the client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	baseURL    string
	httpClient connect.HTTPClient
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *clientConfig) {
		c.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client connect.HTTPClient) ClientOption {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

// NewClient creates a new Email API client.
//
// Example:
//
//	import (
//		emailapi "github.com/emailapi/sdk-go"
//		emailapiv1 "github.com/emailapi/sdk-go/gen/v1"
//		"connectrpc.com/connect"
//	)
//
//	client := emailapi.NewClient("em_...")
//
//	resp, err := client.Emails.SendEmail(ctx, connect.NewRequest(&emailapiv1.SendEmailRequest{
//		From:    "sender@example.com",
//		To:      []string{"recipient@example.com"},
//		Subject: "Hello",
//		Body:    "World",
//	}))
func NewClient(apiKey string, opts ...ClientOption) *Client {
	cfg := &clientConfig{
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Create auth interceptor
	authInterceptor := connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				req.Header().Set("Authorization", "Bearer "+apiKey)
				return next(ctx, req)
			}
		},
	)

	clientOpts := []connect.ClientOption{
		connect.WithInterceptors(authInterceptor),
	}

	return &Client{
		Emails:   emailapiv1connect.NewEmailServiceClient(cfg.httpClient, cfg.baseURL, clientOpts...),
		Domains:  emailapiv1connect.NewDomainServiceClient(cfg.httpClient, cfg.baseURL, clientOpts...),
		Users:    emailapiv1connect.NewUserServiceClient(cfg.httpClient, cfg.baseURL, clientOpts...),
		Webhooks: emailapiv1connect.NewWebhookServiceClient(cfg.httpClient, cfg.baseURL, clientOpts...),
	}
}
