package middleware

import (
	"context"
	"crypto/subtle"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// WebhookInterceptor is a gRPC unary interceptor that validates webhook signatures.
type WebhookInterceptor struct {
	webhookSecret string
}

// NewWebhookInterceptor creates a new webhook verification interceptor.
func NewWebhookInterceptor(webhookSecret string) *WebhookInterceptor {
	return &WebhookInterceptor{
		webhookSecret: webhookSecret,
	}
}

// Unary returns a gRPC unary server interceptor that validates webhook signatures.
func (i *WebhookInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Only verify internal service methods
		if !strings.HasPrefix(info.FullMethod, "/emailapi.v1.InternalService/") {
			return handler(ctx, req)
		}

		// Skip verification for Clerk webhook (uses Svix verification instead)
		if strings.HasSuffix(info.FullMethod, "HandleClerkWebhook") {
			return handler(ctx, req)
		}

		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Get X-Webhook-Secret header
		secretHeaders := md.Get("x-webhook-secret")
		if len(secretHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing x-webhook-secret header")
		}

		providedSecret := secretHeaders[0]
		if providedSecret == "" {
			return nil, status.Error(codes.Unauthenticated, "empty webhook secret")
		}

		// Use constant-time comparison to prevent timing attacks
		if !verifySecret(i.webhookSecret, providedSecret) {
			return nil, status.Error(codes.PermissionDenied, "invalid webhook secret")
		}

		// Secret is valid, proceed with the request
		return handler(ctx, req)
	}
}

// verifySecret performs constant-time comparison of secrets to prevent timing attacks.
func verifySecret(expected, provided string) bool {
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
