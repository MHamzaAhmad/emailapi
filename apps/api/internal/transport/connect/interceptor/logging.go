package interceptor

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
)

// NewLoggingInterceptor creates an interceptor that logs all RPC calls.
func NewLoggingInterceptor(logger zerolog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()

			resp, err := next(ctx, req)

			duration := time.Since(start)
			event := logger.Info()

			if err != nil {
				event = logger.Error().Err(err)
			}

			event.
				Str("procedure", req.Spec().Procedure).
				Str("protocol", req.Peer().Protocol).
				Dur("duration", duration).
				Msg("RPC call")

			return resp, err
		}
	}
}
