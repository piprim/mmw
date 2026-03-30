package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	maxBodyLen = 10000
)

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

// Catch implicit 200 OK responses!
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	i, err := rw.ResponseWriter.Write(b)
	if err != nil {
		return 0, fmt.Errorf("http response writer error: %w", err)
	}

	return i, nil
}

// LoggingMiddleware returns a runner.Middleware that logs every request.
func LoggingMiddleware(logger *slog.Logger, logPayloads bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body []byte
			var err error
			var fields []any

			// Only log the body for POST, PUT, PATCH (where a body is expected)
			if logPayloads &&
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

			if len(body) > 0 {
				// Limit size to avoid log flooding
				l := min(len(body), maxBodyLen)
				fields = append(fields, slog.String("payload", string(body[:l])))
			}

			logger.Log(r.Context(), lvl, "http request handled", fields...)
		})
	}
}
