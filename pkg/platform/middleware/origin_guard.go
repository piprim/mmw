package middleware

import (
	"log/slog"
	"net/http"
)

// OriginGuard returns a Middleware that stops cross-site request forgery on
// requests that change state (every method but GET, HEAD and OPTIONS), which
// matters once the browser sends a session cookie by itself. It refuses a
// request when:
//   - the browser marks it cross-site (Sec-Fetch-Site), or
//   - it carries an Origin that is not in allowedOrigins (a sibling subdomain is
//     same-site, so only this list tells it from our own applications).
//
// A request the browser marks same-origin passes, and so does one with neither
// header (not a browser: it cannot ride on a cookie it was never given). GET
// requests are never checked: they must have no side effect.
func OriginGuard(allowedOrigins []string, logger *slog.Logger) Middleware {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)

				return
			}

			site := r.Header.Get("Sec-Fetch-Site")
			if site == "cross-site" {
				writeOriginRefused(w)

				return
			}

			// The browser itself says the request comes from our own origin.
			if site == "same-origin" {
				next.ServeHTTP(w, r)

				return
			}

			if origin := r.Header.Get("Origin"); origin != "" {
				if _, ok := allowed[origin]; !ok {
					logger.Warn("request refused: origin not allowed", "origin", origin, "path", r.URL.Path)
					writeOriginRefused(w)

					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isSafeMethod reports the methods that must never change state, so a forged
// request cannot do harm through them.
func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func writeOriginRefused(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"code":"permission_denied","message":"request origin not allowed"}`))
}
