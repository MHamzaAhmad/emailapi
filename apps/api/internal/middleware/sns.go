package middleware

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	emailapiv1 "github.com/emailapi/api/gen/v1"
	"github.com/emailapi/api/internal/external/sns"
)

// SNSInterceptor is a gRPC unary interceptor that validates AWS SNS message signatures.
// This middleware performs signature verification before the request reaches the service layer.
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
		if !strings.Contains(info.FullMethod, "SnsService/Handle") {
			return handler(ctx, req)
		}

		// Type assert to SNS request
		snsReq, ok := req.(*emailapiv1.SNSNotificationRequest)
		if !ok {
			// Not an SNS request, pass through
			return handler(ctx, req)
		}

		// Verify the SNS signature
		if err := i.verifier.VerifySignature(
			snsReq.SigningCertUrl,
			snsReq.Signature,
			snsReq.SignatureVersion,
			snsReq.Type,
			snsReq.Message,
			snsReq.MessageId,
			snsReq.Timestamp,
			snsReq.TopicArn,
			snsReq.SubscribeUrl,
			snsReq.Subject,
			snsReq.Token,
		); err != nil {
			// Log the verification failure
			fmt.Printf("SNS signature verification failed in middleware: %v\n", err)
			return nil, status.Errorf(codes.Unauthenticated, "SNS signature verification failed: %v", err)
		}

		// Signature is valid, proceed to handler
		return handler(ctx, req)
	}
}
