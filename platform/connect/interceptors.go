// libs/ogl/platform/connect/interceptors.go
package oglconnect

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
)

// NewErrorLoggingInterceptor returns a Connect interceptor that logs any error
// returned by a handler using eris formatting, preserving the full stack trace.
func NewErrorLoggingInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if err != nil {
				// mapDomainError wraps the original error in a *connect.Error.
				// Unwrap it to recover the eris chain with its stack trace.
				errToLog := err
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					if cause := connectErr.Unwrap(); cause != nil {
						errToLog = cause
					}
				}

				logger.Error("handler error",
					"procedure", req.Spec().Procedure,
					"err", errToLog,
				)
			}

			return resp, err
		}
	}
}
