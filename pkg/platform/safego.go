package platform

import (
	"context"
	"log/slog"

	"github.com/piprim/mmw/platform/middleware"
	"github.com/rotisserie/eris"
)

// SafeGo executes a function in a new goroutine and safely recovers from any panics.
// It prevents a single failing background task from crashing the entire monolith.
// Example:
//
//	// If sendEmail panics, only this email fails.
//	// The panic is logged with the exact same request_id as the HTTP request!
//	platform.SafeGo(ctx, s.logger, func() {
//		s.sendEmail(req.Email)
//	})
func SafeGo(ctx context.Context, logger *slog.Logger, fn func()) {
	go func() {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			// Try to get the Request ID (if this worker was spawned from an HTTP handler)
			reqID := middleware.GetRequestID(ctx)

			// Wrap the panic into eris stack trace
			var err error
			switch v := rec.(type) {
			case error:
				err = eris.Wrap(v, "panic recovered in background worker")
			default:
				err = eris.Errorf("panic recovered in background worker: %v", v)
			}

			// Log the structured error so it triggers your Datadog/Loki alerts
			logger.Error("background worker crashed",
				"request_id", reqID,
				"error", err,
			)
		}()

		// Execute the actual work
		fn()
	}()
}
