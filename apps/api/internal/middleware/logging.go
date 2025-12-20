package middleware

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor creates a gRPC interceptor for logging requests.
type LoggingInterceptor struct {
	logger zerolog.Logger
}

// NewLoggingInterceptor creates a new logging interceptor.
func NewLoggingInterceptor(logger zerolog.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{logger: logger}
}

// Unary returns a gRPC unary server interceptor that logs requests.
func (i *LoggingInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Extract user_id from context if available
		userID, _ := ctx.Value("user_id").(string)

		// Call the handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		logger := i.logger.With().
			Str("method", info.FullMethod).
			Dur("duration", duration).
			Str("user_id", userID).
			Logger()

		if err != nil {
			st, _ := status.FromError(err)
			logger.Error().
				Str("error", err.Error()).
				Str("code", st.Code().String()).
				Msg("gRPC request failed")
		} else {
			logger.Info().Msg("gRPC request completed")
		}

		return resp, err
	}
}
