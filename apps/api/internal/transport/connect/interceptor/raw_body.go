package interceptor

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"connectrpc.com/connect"
)

// Context key for storing raw body bytes.
type rawBodyKey struct{}

// GetRawBody retrieves the raw request body from context.
// Returns nil if no raw body was stored.
func GetRawBody(ctx context.Context) []byte {
	if raw, ok := ctx.Value(rawBodyKey{}).([]byte); ok {
		return raw
	}
	return nil
}

// NewRawBodyInterceptor creates an interceptor that captures raw request bodies
// for procedures that need signature verification (e.g., webhooks).
//
// The raw body is stored in the request context and can be retrieved using GetRawBody.
func NewRawBodyInterceptor() connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			// Only capture raw body for webhook endpoints that need signature verification
			needsRawBody := procedure == "/v1.InternalService/HandleClerkWebhook" ||
				procedure == "/v1.InternalService/HandlePolarWebhook" ||
				procedure == "/v1.SnsService/HandleSNSNotification"

			if !needsRawBody {
				return next(ctx, req)
			}

			// Access the underlying HTTP request
			// Connect provides this via the request's peer info, but we need to intercept
			// at the HTTP layer. Since we can't easily access the raw body from here,
			// we need a different approach - use an HTTP middleware instead.

			// For now, let's pass through and handle this via HTTP middleware
			return next(ctx, req)
		}
	}
	return interceptor
}

// RawBodyHTTPMiddleware is an HTTP middleware that captures raw request bodies
// for specific paths and stores them in the request context.
func RawBodyHTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only capture for webhook endpoints
		needsRawBody := r.URL.Path == "/v1.InternalService/HandleClerkWebhook" ||
			r.URL.Path == "/v1.InternalService/HandlePolarWebhook" ||
			r.URL.Path == "/v1.SnsService/HandleSNSNotification"

		if needsRawBody && r.Body != nil {
			// Read the body
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}

			// Restore the body for downstream handlers (Connect needs to read it too)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// Store raw bytes in context
			ctx := context.WithValue(r.Context(), rawBodyKey{}, bodyBytes)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
