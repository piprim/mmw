package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	pfauthctx "github.com/piprim/mmw/pkg/platform/authctx"
)

// AccessChecker tells whether the user may access application. Implementations
// are provided by the application that owns the access data, typically as a
// closure over one of its queries. An error means the answer is unknown.
type AccessChecker func(ctx context.Context, userID uuid.UUID, application string) (bool, error)

// AppAuthorizationMiddleware returns a Middleware that lets a request through
// only if the authenticated user may access application, as decided by check.
// It must run after BearerAuthMiddleware, which tells it who the user is.
//
// A request with no authenticated user gets HTTP 401; a user who may not access
// application gets HTTP 403; a check that fails gets HTTP 503, so that a
// failure never lets a request through. Bodies are Connect-compatible JSON.
func AppAuthorizationMiddleware(application string, check AccessChecker, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := pfauthctx.UserIDFromContext(r.Context())
			if err != nil {
				writeUnauthorized(w)

				return
			}

			allowed, err := check(r.Context(), userID, application)
			if err != nil {
				logger.Error("application access check failed", "err", err, "user_id", userID, "application", application)
				writeUnavailable(w)

				return
			}

			if !allowed {
				logger.Warn("application access denied", "user_id", userID, "application", application, "path", r.URL.Path)
				writeForbidden(w)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"code":"permission_denied","message":"access to this application is not allowed"}`))
}

func writeUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"code":"unavailable","message":"cannot check access to this application"}`))
}
