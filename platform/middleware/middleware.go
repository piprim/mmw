package middleware

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	connectcors "connectrpc.com/cors"
	"github.com/google/uuid"
	"github.com/ovya/ogl/platform"
	"github.com/rs/cors"
)

const maxAge = 7200 // Unit is second

// Middleware defines a standard HTTP middleware constructor
type Middleware func(http.Handler) http.Handler

// responseWriter is a minimal wrapper to capture the HTTP status code
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

type contextKey int

const (
	// requestIDKey is the unique key for the request ID in the context.
	// We use an unexported type to avoid collisions with other packages.
	requestIDKey contextKey = iota
)

// LoggingMiddleware returns a runner.Middleware that logs every request.
func LoggingMiddleware(logger *slog.Logger, withPayload bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body []byte
			var err error
			var fields []any

			// Only log the body for POST, PUT, PATCH (where a body is expected)
			if withPayload &&
				(r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
				body, err = io.ReadAll(r.Body)
				if err != nil {
					logger.Error("failed to read request body for logging", "error", err)
				} else {
					// IMPORTANT: Restore the body so the next handler can read it!
					r.Body = io.NopCloser(bytes.NewBuffer(body))
				}
			}

			start := time.Now()

			// Generate or extract a Request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			w.Header().Set("X-Request-ID", requestID)
			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			next.ServeHTTP(wrapped, r.WithContext(ctx))
			duration := time.Since(start)

			// Use appropriate log level based on status code
			lvl := slog.LevelInfo
			if wrapped.status >= http.StatusInternalServerError {
				lvl = slog.LevelError
			} else if wrapped.status >= http.StatusNotFound {
				lvl = slog.LevelWarn
			}

			fields = []any{
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.status,
				"duration", duration,
				"ip", r.RemoteAddr,
			}

			// Add the body to logs if it exists and isn't too large
			if len(body) > 0 && len(body) < 10000 { // Limit size to avoid log flooding
				fields = append(fields, slog.String("payload", string(body)))
			}

			logger.Log(r.Context(), lvl, "http request handled", fields...)
		})
	}
}

// CORSMiddleware adds CORS support for Connect, gRPC, and gRPC-Web
func CORSMiddleware(conf platform.Config) Middleware {
	allowedOrigins := "*"
	if conf.GetAppEnv().String() != "development" {
		allowedOrigins = conf.GetServerHost()
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{allowedOrigins},
		// The official Connect helpers to inject all required gRPC/Connect headers
		// See the documentation: https://pkg.go.dev/connectrpc.com/cors#section-readme
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         maxAge, // Optional cache preflight requests
	})

	return func(next http.Handler) http.Handler {
		return c.Handler(next)
	}
}
