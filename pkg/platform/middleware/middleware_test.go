package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	pfconfig "github.com/piprim/mmw/pkg/platform/config"
	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware(t *testing.T) {
	t.Run("logs request details and sets X-Request-ID header", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		mw := LoggingMiddleware(logger, true)
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("request body"))
		rec := httptest.NewRecorder()
		mw(nextHandler).ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
		assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))

		logOutput := buf.String()
		assert.Contains(t, logOutput, "level=INFO")
		assert.Contains(t, logOutput, "msg=\"http request handled\"")
		assert.Contains(t, logOutput, "method=POST")
		assert.Contains(t, logOutput, "path=/test")
		assert.Contains(t, logOutput, "status=200")
		assert.Contains(t, logOutput, "payload=\"request body\"")
	})
}

func TestCORSMiddleware(t *testing.T) {
	t.Run("development: no Access-Control-Allow-Origin for arbitrary origin", func(t *testing.T) {
		cfg := &pfconfig.Server{}
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		mw := CORSMiddleware(cfg)
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://any-origin.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		mw(nextHandler).ServeHTTP(rec, req)

		assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("production: allows configured origin and blocks others", func(t *testing.T) {
		cfg := &pfconfig.Server{Host: "https://api.example.com"}
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		mw := CORSMiddleware(cfg)

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://api.example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		mw(nextHandler).ServeHTTP(rec, req)
		assert.Equal(t, "https://api.example.com", rec.Header().Get("Access-Control-Allow-Origin"))

		reqBad := httptest.NewRequest(http.MethodOptions, "/test", nil)
		reqBad.Header.Set("Origin", "https://evil.com")
		reqBad.Header.Set("Access-Control-Request-Method", "POST")
		recBad := httptest.NewRecorder()
		mw(nextHandler).ServeHTTP(recBad, reqBad)
		assert.Empty(t, recBad.Header().Get("Access-Control-Allow-Origin"))
	})
}
