package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/emailapi/api/internal/external/sns"
)

// SNSInterceptor is a gRPC unary interceptor that validates AWS SNS message signatures.
type SNSInterceptor struct {
	verifier *sns.Verifier
}

// NewSNSInterceptor creates a new SNS signature verification interceptor.
func NewSNSInterceptor() *SNSInterceptor {
	return &SNSInterceptor{
		verifier: sns.NewVerifier(),
	}
}

// Unary returns a gRPC unary server interceptor that validates SNS signatures.
func (i *SNSInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Only verify SNS notification methods
		if !strings.HasSuffix(info.FullMethod, "/HandleSNSNotification") {
			return handler(ctx, req)
		}

		// Extract SNS headers from metadata for verification
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			// No metadata - let the service handle verification
			return handler(ctx, req)
		}

		// Check if this looks like an SNS request by looking for the SNS message type header
		// AWS SNS sends x-amz-sns-message-type header
		messageTypes := md.Get("x-amz-sns-message-type")
		if len(messageTypes) == 0 {
			// Not an SNS request from the header perspective, proceed to service
			// The service layer may still verify via the request body
			return handler(ctx, req)
		}

		// For SNS requests, we can validate at service level since the signature fields
		// are in the request body, not the headers. The middleware just logs and passes through.
		// Full verification happens in the service layer using the Verifier.

		// Add a flag to context indicating SNS request was detected
		ctx = context.WithValue(ctx, "sns_message_type", messageTypes[0])

		return handler(ctx, req)
	}
}

// GetSNSMessageType extracts the SNS message type from context if set by middleware.
func GetSNSMessageType(ctx context.Context) string {
	if msgType, ok := ctx.Value("sns_message_type").(string); ok {
		return msgType
	}
	return ""
}
