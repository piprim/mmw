package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	pfauthctx "github.com/piprim/mmw/pkg/platform/authctx"
	"github.com/piprim/mmw/pkg/platform/sessioncookie"
)

// RenewedTokenHeader is the response header carrying a renewed token to a Bearer client.
// A browser session never gets it: its renewed token travels in the session cookie.
const RenewedTokenHeader = "X-Renewed-Token"

// TokenValidation is what a TokenValidator learns about a valid bearer token.
type TokenValidation struct {
	// UserID is the authenticated user.
	UserID uuid.UUID
	// RenewedToken, when not empty, replaces the validated token: it had expired
	// but could be renewed. The middleware hands it back in the session cookie when the
	// request came with it, in RenewedTokenHeader when it came as a Bearer.
	RenewedToken string
}

// TokenValidator validates a bearer token and returns who it authenticates.
// Implementations are provided by the caller, typically as a closure over an
// AuthPrivateService (see the todo module's NewTokenValidator adapter).
type TokenValidator func(ctx context.Context, token string) (TokenValidation, error)

// AuthOption customises AuthMiddleware.
type AuthOption func(*authOptions)

type authOptions struct {
	cookie *sessioncookie.Cookie
}

// WithSessionCookie makes the middleware accept the token from the session
// cookie, ahead of the Authorization header, and hand a renewed token back in
// that cookie when the request came with it.
func WithSessionCookie(c *sessioncookie.Cookie) AuthOption {
	return func(o *authOptions) { o.cookie = c }
}

// BearerAuthMiddleware is AuthMiddleware that reads the token from the Bearer
// Authorization header only.
func BearerAuthMiddleware(validate TokenValidator, logger *slog.Logger, excludedPaths []string) Middleware {
	return AuthMiddleware(validate, logger, excludedPaths)
}

// AuthMiddleware returns a Middleware that enforces token authentication.
//
// For each request it:
//  1. Skips auth for paths in excludedPaths (or any /debug/ path).
//  2. Extracts the token: from the session cookie when WithSessionCookie is
//     given and the cookie is present, otherwise from the Bearer Authorization header.
//  3. Calls validate; on success it injects the userID via pfauthctx.WithUserID and,
//     if the token was renewed, sets it back in the session cookie when the token came
//     from it, or in RenewedTokenHeader when it came as a Bearer.
//  4. Returns HTTP 401 with a Connect-compatible JSON body on any failure.
func AuthMiddleware(
	validate TokenValidator, logger *slog.Logger, excludedPaths []string, opts ...AuthOption,
) Middleware {
	var o authOptions
	for _, opt := range opts {
		opt(&o)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, path := range excludedPaths {
				if strings.HasPrefix(r.URL.Path, path) || strings.Contains(r.URL.Path, "/debug/") {
					next.ServeHTTP(w, r)

					return
				}
			}

			token, fromCookie := o.extractToken(r)
			if token == "" {
				writeUnauthorized(w)

				return
			}

			validation, err := validate(r.Context(), token)
			if err != nil {
				logger.Error("token validation failed", "err", err, "path", r.URL.Path)
				writeUnauthorized(w)

				return
			}

			if validation.RenewedToken != "" {
				if fromCookie {
					o.cookie.Set(w.Header(), validation.RenewedToken)
				} else {
					w.Header().Set(RenewedTokenHeader, validation.RenewedToken)
				}
			}

			ctx := pfauthctx.WithUserID(r.Context(), validation.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken returns the request token and whether it came from the session
// cookie, which then also receives the renewed token.
func (o authOptions) extractToken(r *http.Request) (token string, fromCookie bool) {
	if o.cookie != nil {
		if token = o.cookie.Token(r.Header); token != "" {
			return token, true
		}
	}

	return extractBearerToken(r), false
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
