package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	pfconfig "github.com/piprim/mmw/platform/config"
	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware(t *testing.T) {
	// Setup logger to capture output
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	// Create a simple handler that the middleware will wrap
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create middleware
	mw := LoggingMiddleware(logger, true)
	wrappedHandler := mw(nextHandler)

	// Create a request
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("request body"))
	rec := httptest.NewRecorder()

	// Execute
	wrappedHandler.ServeHTTP(rec, req)

	// Verify Response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())

	// Verify Header X-Request-ID
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))

	// Verify Log Output
	logOutput := buf.String()
	assert.Contains(t, logOutput, "level=INFO")
	assert.Contains(t, logOutput, "msg=\"http request handled\"")
	assert.Contains(t, logOutput, "method=POST")
	assert.Contains(t, logOutput, "path=/test")
	assert.Contains(t, logOutput, "status=200")
	assert.Contains(t, logOutput, "payload=\"request body\"")
}

func TestCORSMiddleware_Development(t *testing.T) {
	cfg := &pfconfig.Server{}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := CORSMiddleware(cfg)
	wrappedHandler := mw(nextHandler)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// In development, allowed origins should be "*" (or reflected)
	assert.Equal(t, "", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_Production(t *testing.T) {
	cfg := &pfconfig.Server{
		Host: "https://api.example.com",
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := CORSMiddleware(cfg)
	wrappedHandler := mw(nextHandler)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://api.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	assert.Equal(t, "https://api.example.com", rec.Header().Get("Access-Control-Allow-Origin"))

	// Test disallowed origin
	reqBad := httptest.NewRequest(http.MethodOptions, "/test", nil)
	reqBad.Header.Set("Origin", "https://evil.com")
	reqBad.Header.Set("Access-Control-Request-Method", "POST")
	recBad := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recBad, reqBad)
	assert.Empty(t, recBad.Header().Get("Access-Control-Allow-Origin"))
}
