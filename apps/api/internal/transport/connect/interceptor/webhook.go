package interceptor

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
)

// WebhookConfig holds configuration for the webhook interceptor.
type WebhookConfig struct {
	InternalWebhookSecret string
}

// NewWebhookInterceptor creates an interceptor that validates internal webhook secrets.
func NewWebhookInterceptor(cfg WebhookConfig) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			// Only check for InternalService endpoints (except Clerk webhook which uses Svix)
			if !strings.HasPrefix(procedure, "/emailapi.v1.InternalService/") {
				return next(ctx, req)
			}

			// Skip webhook secret check for Clerk (uses Svix) and Polar (uses Standard Webhooks)
			if strings.Contains(procedure, "HandleClerkWebhook") || strings.Contains(procedure, "HandlePolarWebhook") {
				return next(ctx, req)
			}

			// Verify webhook secret
			webhookSecret := req.Header().Get("X-Webhook-Secret")
			if webhookSecret == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing X-Webhook-Secret header"))
			}

			if webhookSecret != cfg.InternalWebhookSecret {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid webhook secret"))
			}

			return next(ctx, req)
		}
	}
}
