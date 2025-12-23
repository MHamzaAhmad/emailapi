package middleware

import (
	"context"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/emailapi/api/internal/service"
)

// AuthMethod indicates how the request was authenticated.
type AuthMethod string

const (
	AuthMethodClerk  AuthMethod = "clerk"
	AuthMethodAPIKey AuthMethod = "api_key"
)

// Context keys for auth information.
type contextKey string

const (
	ContextKeyUserID     contextKey = "user_id"
	ContextKeyAuthMethod contextKey = "auth_method"
	ContextKeyAPIKeyID   contextKey = "api_key_id"
)

// UserLookup interface for looking up users by external ID.
type UserLookup interface {
	GetByExternalID(ctx context.Context, externalID string) (userID string, err error)
}

// AuthInterceptor is a gRPC unary interceptor that validates Clerk JWTs and API keys.
type AuthInterceptor struct {
	apiKeyService  *service.APIKeyService
	userLookup     UserLookup
	clerkSecretKey string
	publicMethods  map[string]bool
	apiKeyAllowed  map[string]bool // Services that allow API key auth in addition to Clerk
}

// AuthInterceptorConfig holds configuration for the auth interceptor.
type AuthInterceptorConfig struct {
	APIKeyService  *service.APIKeyService
	UserLookup     UserLookup
	ClerkSecretKey string
}

// NewAuthInterceptor creates a new auth interceptor with Clerk JWT + API key support.
func NewAuthInterceptor(cfg AuthInterceptorConfig) *AuthInterceptor {
	return &AuthInterceptor{
		apiKeyService:  cfg.APIKeyService,
		userLookup:     cfg.UserLookup,
		clerkSecretKey: cfg.ClerkSecretKey,
		publicMethods:  map[string]bool{
			// CreateUser is no longer public - users are created via Clerk webhook
		},
		apiKeyAllowed: map[string]bool{
			// EmailService and DomainService allow API key auth for programmatic access
			"/emailapi.v1.EmailService/SendEmail":        true,
			"/emailapi.v1.EmailService/SendEmailAsync":   true,
			"/emailapi.v1.EmailService/GetEmail":         true,
			"/emailapi.v1.DomainService/CreateDomain":    true,
			"/emailapi.v1.DomainService/GetDomain":       true,
			"/emailapi.v1.DomainService/ListDomains":     true,
			"/emailapi.v1.DomainService/VerifyDomain":    true,
			"/emailapi.v1.DomainService/DeleteDomain":    true,
			"/emailapi.v1.DomainService/RefreshDNS":      true,
			"/emailapi.v1.DomainService/SetMailFrom":     true,
			"/emailapi.v1.DomainService/GetRoutingRules": true,
			"/emailapi.v1.DomainService/SetRoutingRules": true,
		},
	}
}

// Unary returns a gRPC unary server interceptor that validates auth.
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for InternalService - uses webhook secret verification
		if strings.HasPrefix(info.FullMethod, "/emailapi.v1.InternalService/") {
			return handler(ctx, req)
		}

		// Skip auth for SnsService - uses SNS signature verification
		if strings.HasPrefix(info.FullMethod, "/emailapi.v1.SnsService/") {
			return handler(ctx, req)
		}

		// Check if this is a public method
		if i.publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract metadata from context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Get authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Extract token from "Bearer <token>"
		authHeader := authHeaders[0]
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "empty authorization token")
		}

		// Determine if token looks like an API key (starts with em_)
		isAPIKey := strings.HasPrefix(token, "em_")

		if isAPIKey {
			// Check if this endpoint allows API key auth
			if !i.apiKeyAllowed[info.FullMethod] {
				return nil, status.Error(codes.Unauthenticated, "this endpoint requires Clerk authentication, API keys not allowed")
			}

			// Validate API key
			user, apiKey, err := i.apiKeyService.ValidateAndGetUser(ctx, token)
			if err != nil {
				return nil, status.Errorf(codes.Unauthenticated, "invalid API key: %v", err)
			}

			// Add auth info to context
			ctx = context.WithValue(ctx, ContextKeyUserID, user.ID)
			ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodAPIKey)
			ctx = context.WithValue(ctx, ContextKeyAPIKeyID, apiKey.ID)

			return handler(ctx, req)
		}

		// Try Clerk JWT verification
		claims, err := jwt.Verify(ctx, &jwt.VerifyParams{
			Token: token,
		})
		if err != nil {
			// If Clerk verification fails AND this endpoint allows API keys,
			// try API key auth as fallback (in case token format detection failed)
			if i.apiKeyAllowed[info.FullMethod] {
				user, apiKey, apiKeyErr := i.apiKeyService.ValidateAndGetUser(ctx, token)
				if apiKeyErr == nil {
					ctx = context.WithValue(ctx, ContextKeyUserID, user.ID)
					ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodAPIKey)
					ctx = context.WithValue(ctx, ContextKeyAPIKeyID, apiKey.ID)
					return handler(ctx, req)
				}
			}
			return nil, status.Errorf(codes.Unauthenticated, "invalid Clerk token: %v", err)
		}

		// Extract Clerk user ID (subject claim)
		clerkUserID := claims.Subject
		if clerkUserID == "" {
			return nil, status.Error(codes.Unauthenticated, "missing subject in Clerk token")
		}

		// Look up internal user by Clerk external_id
		userID, err := i.userLookup.GetByExternalID(ctx, clerkUserID)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "user not found for Clerk ID %s: %v", clerkUserID, err)
		}

		// Add auth info to context
		ctx = context.WithValue(ctx, ContextKeyUserID, userID)
		ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodClerk)

		return handler(ctx, req)
	}
}

// GetUserID extracts the user ID from the context.
// Returns empty string if not found.
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(ContextKeyUserID).(string); ok {
		return userID
	}
	// Backwards compatibility with old "user_id" key
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// GetAuthMethod extracts the auth method from the context.
func GetAuthMethod(ctx context.Context) AuthMethod {
	if method, ok := ctx.Value(ContextKeyAuthMethod).(AuthMethod); ok {
		return method
	}
	return ""
}

// GetAPIKeyID extracts the API key ID from the context (only set for API key auth).
func GetAPIKeyID(ctx context.Context) string {
	if apiKeyID, ok := ctx.Value(ContextKeyAPIKeyID).(string); ok {
		return apiKeyID
	}
	// Backwards compatibility
	if apiKeyID, ok := ctx.Value("api_key_id").(string); ok {
		return apiKeyID
	}
	return ""
}

// InitClerk initializes the global Clerk client with the secret key.
// This must be called before using Clerk JWT verification.
func InitClerk(secretKey string) {
	clerk.SetKey(secretKey)
}
