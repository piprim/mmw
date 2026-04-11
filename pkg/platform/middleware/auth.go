package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	pfauthctx "github.com/piprim/mmw/pkg/platform/authctx"
)

// TokenValidator validates a bearer token and returns the authenticated user's UUID.
// Implementations are provided by the caller, typically as a closure over an
// AuthPrivateService (see the todo module's NewTokenValidator adapter).
type TokenValidator func(ctx context.Context, token string) (uuid.UUID, error)

// BearerAuthMiddleware returns a Middleware that enforces bearer-token authentication.
//
// For each request it:
//  1. Skips auth for paths in excludedPaths (or any /debug/ path).
//  2. Extracts the Bearer token from the Authorization header.
//  3. Calls validate; on success it injects the userID via pfauthctx.WithUserID.
//  4. Returns HTTP 401 with a Connect-compatible JSON body on any failure.
func BearerAuthMiddleware(validate TokenValidator, logger *slog.Logger, excludedPaths []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, path := range excludedPaths {
				if strings.HasPrefix(r.URL.Path, path) || strings.Contains(r.URL.Path, "/debug/") {
					next.ServeHTTP(w, r)
					return
				}
			}

			token := extractBearerToken(r)
			if token == "" {
				writeUnauthorized(w)
				return
			}

			userID, err := validate(r.Context(), token)
			if err != nil {
				logger.Error("token validation failed", "err", err, "path", r.URL.Path)
				writeUnauthorized(w)
				return
			}

			ctx := pfauthctx.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}

	return strings.TrimPrefix(auth, "Bearer ")
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"code":"unauthenticated","message":"missing or invalid token"}`))
}
