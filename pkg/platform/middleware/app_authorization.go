package middleware

import (
	"log/slog"
	"net/http"
	"slices"

	pfauthctx "github.com/piprim/mmw/pkg/platform/authctx"
)

// AppAuthorizationMiddleware returns a Middleware that lets a request through
// only if the authenticated user may access application. It must run after
// BearerAuthMiddleware, which tells it who the user is and what they may access.
//
// A request with no authenticated user gets HTTP 401; a user who may not access
// application gets HTTP 403, with a Connect-compatible JSON body.
func AppAuthorizationMiddleware(application string, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := pfauthctx.UserIDFromContext(r.Context())
			if err != nil {
				writeUnauthorized(w)

				return
			}

			if !slices.Contains(pfauthctx.ApplicationsFromContext(r.Context()), application) {
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
