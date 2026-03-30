package middleware

import (
	"log/slog"
	"net/http"

	"github.com/rotisserie/eris"
)

// RecoveryMiddleware catches panics, logs the stack trace, and returns a 500 status.
func RecoveryMiddleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Defer the recovery function to run when the handler finishes (or panics)
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				// Grab the Request ID so we can correlate this crash
				// with the earlier "request started" log.
				reqID := GetRequestID(r.Context())

				// Convert the panic to an eris error to capture the stack trace
				var err error
				switch v := rec.(type) {
				case error:
					err = eris.Wrap(v, "unhandled panic in http handler")
				default:
					err = eris.Errorf("unhandled panic in http handler: %v", v)
				}

				// Log it!
				logger.Error("server panic recovered",
					"request_id", reqID,
					"method", r.Method,
					"path", r.URL.Path,
					"error", err,
				)

				// 5. Return a safe, generic error to the client
				// We DO NOT send the stack trace to the user for security reasons.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error": "Internal Server Error"}`))
			}()

			// Execute the actual request
			next.ServeHTTP(w, r)
		})
	}
}
