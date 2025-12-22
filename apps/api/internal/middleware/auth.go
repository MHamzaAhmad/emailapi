package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/emailapi/api/internal/service"
)

// AuthInterceptor is a gRPC unary interceptor that validates API keys.
type AuthInterceptor struct {
	apiKeyService *service.APIKeyService
	// Public methods that don't require authentication
	publicMethods map[string]bool
}

// NewAuthInterceptor creates a new auth interceptor.
func NewAuthInterceptor(apiKeyService *service.APIKeyService) *AuthInterceptor {
	return &AuthInterceptor{
		apiKeyService: apiKeyService,
		publicMethods: map[string]bool{
			"/emailapi.v1.UserService/CreateUser": true, // Allow user creation without auth
		},
	}
}

// Unary returns a gRPC unary server interceptor that validates API keys.
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
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

		// Validate API key and get user
		user, apiKey, err := i.apiKeyService.ValidateAndGetUser(ctx, token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid API key: %v", err)
		}

		// Add user_id to context
		ctx = context.WithValue(ctx, "user_id", user.ID)
		ctx = context.WithValue(ctx, "api_key_id", apiKey.ID)

		// Call the handler with the new context
		return handler(ctx, req)
	}
}

// HTTPMiddleware returns an HTTP middleware that copies Authorization header to gRPC metadata.
func HTTPMiddleware() func(next grpc.UnaryHandler) grpc.UnaryHandler {
	return func(next grpc.UnaryHandler) grpc.UnaryHandler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// This is handled by the gateway automatically
			// The gateway copies HTTP headers to gRPC metadata
			return next(ctx, req)
		}
	}
}

// MetadataAnnotator is used by grpc-gateway to copy HTTP headers to gRPC metadata.
func MetadataAnnotator(ctx context.Context, req interface{}) metadata.MD {
	md := metadata.MD{}

	// The grpc-gateway already handles this automatically for most headers
	// This is just for custom headers if needed

	return md
}

// GetUserID extracts the user ID from the context.
// Returns empty string if not found.
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}
