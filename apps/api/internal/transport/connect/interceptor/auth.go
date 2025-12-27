package interceptor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/clerk/clerk-sdk-go/v2/jwt"

	"github.com/emailapi/api/internal/domain"
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
	ContextKeyAPIKey     contextKey = "api_key"
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

// apiKeyAllowedProcedures defines which procedures allow API key authentication.
var apiKeyAllowedProcedures = map[string]bool{
	"/v1.EmailService/SendEmail":     true,
	"/v1.EmailService/StreamEvents":  true,
	"/v1.DomainService/AddDomain":    true,
	"/v1.DomainService/GetDomain":    true,
	"/v1.DomainService/ListDomains":  true,
	"/v1.DomainService/VerifyDomain": true,
	"/v1.DomainService/DeleteDomain": true,
}

// requiredScopes maps procedures to their required scopes.
// API key must have ALL listed scopes to access the procedure.
var requiredScopes = map[string][]domain.Scope{
	"/v1.EmailService/SendEmail":     {domain.ScopeEmailSend},
	"/v1.EmailService/StreamEvents":  {domain.ScopeEmailSend}, // Use same scope as send for streaming events
	"/v1.DomainService/AddDomain":    {domain.ScopeDomainWrite},
	"/v1.DomainService/GetDomain":    {domain.ScopeDomainRead},
	"/v1.DomainService/ListDomains":  {domain.ScopeDomainRead},
	"/v1.DomainService/VerifyDomain": {domain.ScopeDomainWrite},
	"/v1.DomainService/DeleteDomain": {domain.ScopeDomainWrite},
}

// checkAPIKeyScopes verifies the API key has all required scopes for the procedure.
func checkAPIKeyScopes(apiKey *domain.APIKey, procedure string) error {
	scopes, ok := requiredScopes[procedure]
	if !ok {
		// No scopes required for this procedure
		return nil
	}

	for _, required := range scopes {
		if !apiKey.HasScope(required) {
			return fmt.Errorf("API key missing required scope: %s", required)
		}
	}
	return nil
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

// authInterceptor implements the full connect.Interceptor interface for both unary and streaming.
type authInterceptor struct {
	cfg AuthConfig
}

// NewCombinedAuthInterceptor creates an interceptor that handles both unary and streaming calls.
func NewCombinedAuthInterceptor(cfg AuthConfig) connect.Interceptor {
	return &authInterceptor{cfg: cfg}
}

// WrapUnary implements connect.Interceptor for unary calls.
func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		procedure := req.Spec().Procedure

		// Skip auth for InternalService - uses webhook secret verification
		if strings.HasPrefix(procedure, "/v1.InternalService/") {
			return next(ctx, req)
		}

		// Skip auth for SnsService - uses SNS signature verification
		if strings.HasPrefix(procedure, "/v1.SnsService/") {
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

		// Authenticate and get user context
		authCtx, err := authenticateToken(ctx, a.cfg, procedure, token)
		if err != nil {
			return nil, err
		}

		return next(authCtx, req)
	}
}

// WrapStreamingClient implements connect.Interceptor (no-op for server).
func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next // Pass-through for client streaming (not used on server)
}

// WrapStreamingHandler implements connect.Interceptor for server streaming calls.
func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		procedure := conn.Spec().Procedure

		// Skip auth for InternalService
		if strings.HasPrefix(procedure, "/v1.InternalService/") {
			return next(ctx, conn)
		}

		// Skip auth for SnsService
		if strings.HasPrefix(procedure, "/v1.SnsService/") {
			return next(ctx, conn)
		}

		// Check if this is a public procedure
		if publicProcedures[procedure] {
			return next(ctx, conn)
		}

		// Extract authorization header
		authHeader := conn.RequestHeader().Get("Authorization")
		if authHeader == "" {
			return connect.NewError(connect.CodeUnauthenticated, errors.New("missing authorization header"))
		}

		// Extract token from "Bearer <token>"
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return connect.NewError(connect.CodeUnauthenticated, errors.New("invalid authorization header format"))
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			return connect.NewError(connect.CodeUnauthenticated, errors.New("empty authorization token"))
		}

		// Authenticate and get user context
		authCtx, err := authenticateToken(ctx, a.cfg, procedure, token)
		if err != nil {
			return err
		}

		return next(authCtx, conn)
	}
}

// authenticateToken validates a token (API key or Clerk JWT) and returns an authenticated context.
func authenticateToken(ctx context.Context, cfg AuthConfig, procedure, token string) (context.Context, error) {
	// Determine if token looks like an API key (starts with sea_live_, sea_test_)
	isAPIKey := strings.HasPrefix(token, "sea_live_") || strings.HasPrefix(token, "sea_test_")

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

		// Check API key scopes
		if err := checkAPIKeyScopes(apiKey, procedure); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}

		// Add auth info to context
		ctx = context.WithValue(ctx, ContextKeyUserID, user.ID)
		ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodAPIKey)
		ctx = context.WithValue(ctx, ContextKeyAPIKeyID, apiKey.ID)
		ctx = context.WithValue(ctx, ContextKeyAPIKey, apiKey)

		return ctx, nil
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
				// Check API key scopes
				if scopeErr := checkAPIKeyScopes(apiKey, procedure); scopeErr != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, scopeErr)
				}
				ctx = context.WithValue(ctx, ContextKeyUserID, user.ID)
				ctx = context.WithValue(ctx, ContextKeyAuthMethod, AuthMethodAPIKey)
				ctx = context.WithValue(ctx, ContextKeyAPIKeyID, apiKey.ID)
				ctx = context.WithValue(ctx, ContextKeyAPIKey, apiKey)
				return ctx, nil
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

	return ctx, nil
}
