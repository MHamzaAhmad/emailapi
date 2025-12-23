package interceptor

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/clerk/clerk-sdk-go/v2/jwt"

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

// AuthConfig holds configuration for the auth interceptor.
type AuthConfig struct {
	APIKeyService  *service.APIKeyService
	UserLookup     UserLookup
	ClerkSecretKey string
}

// publicProcedures are procedures that don't require authentication.
var publicProcedures = map[string]bool{
	// Currently none - add procedures here if needed
}

// apiKeyAllowedProcedures are procedures that allow API key auth.
var apiKeyAllowedProcedures = map[string]bool{
	"/emailapi.v1.EmailService/SendEmail":     true,
	"/emailapi.v1.DomainService/AddDomain":    true,
	"/emailapi.v1.DomainService/GetDomain":    true,
	"/emailapi.v1.DomainService/ListDomains":  true,
	"/emailapi.v1.DomainService/VerifyDomain": true,
	"/emailapi.v1.DomainService/DeleteDomain": true,
}

// NewAuthInterceptor creates an interceptor that validates Clerk JWTs and API keys.
func NewAuthInterceptor(cfg AuthConfig) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			// Skip auth for InternalService - uses webhook secret verification
			if strings.HasPrefix(procedure, "/emailapi.v1.InternalService/") {
				return next(ctx, req)
			}

			// Skip auth for SnsService - uses SNS signature verification
			if strings.HasPrefix(procedure, "/emailapi.v1.SnsService/") {
				return next(ctx, req)
			}

			// Check if this is a public procedure
			if publicProcedures[procedure] {
				return next(ctx, req)
			}

			// Extract authorization header
			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing authorization header"))
			}

			// Extract token from "Bearer <token>"
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid authorization header format"))
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("empty authorization token"))
			}

			// Determine if token looks like an API key (starts with em_)
			isAPIKey := strings.HasPrefix(token, "em_")

			if isAPIKey {
				// Check if this endpoint allows API key auth
				if !apiKeyAllowedProcedures[procedure] {
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("this endpoint requires Clerk authentication, API keys not allowed"))
				}

				// Validate API key
				user, apiKey, err := cfg.APIKeyService.ValidateAndGetUser(ctx, token)
				if err != nil {
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid API key"))
				}

				// Add auth info to context
				ctx = context.WithValue(ctx, ContextKeyUserID, user.ID)
				ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodAPIKey)
				ctx = context.WithValue(ctx, ContextKeyAPIKeyID, apiKey.ID)

				return next(ctx, req)
			}

			// Try Clerk JWT verification
			claims, err := jwt.Verify(ctx, &jwt.VerifyParams{
				Token: token,
			})
			if err != nil {
				// If Clerk verification fails AND this endpoint allows API keys,
				// try API key auth as fallback
				if apiKeyAllowedProcedures[procedure] {
					user, apiKey, apiKeyErr := cfg.APIKeyService.ValidateAndGetUser(ctx, token)
					if apiKeyErr == nil {
						ctx = context.WithValue(ctx, ContextKeyUserID, user.ID)
						ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodAPIKey)
						ctx = context.WithValue(ctx, ContextKeyAPIKeyID, apiKey.ID)
						return next(ctx, req)
					}
				}
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid Clerk token"))
			}

			// Extract Clerk user ID (subject claim)
			clerkUserID := claims.Subject
			if clerkUserID == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing subject in Clerk token"))
			}

			// Look up internal user by Clerk external_id
			userID, err := cfg.UserLookup.GetByExternalID(ctx, clerkUserID)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not found for Clerk ID"))
			}

			// Add auth info to context
			ctx = context.WithValue(ctx, ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodClerk)

			return next(ctx, req)
		}
	}
}

// GetUserID extracts the user ID from the context.
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

// GetAPIKeyID extracts the API key ID from the context.
func GetAPIKeyID(ctx context.Context) string {
	if apiKeyID, ok := ctx.Value(ContextKeyAPIKeyID).(string); ok {
		return apiKeyID
	}
	return ""
}
