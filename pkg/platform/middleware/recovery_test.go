package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mw := RecoveryMiddleware(logger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	mw(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestRecoveryMiddleware_PanicWithError(t *testing.T) {
	var buf nopWriter
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})

	mw := RecoveryMiddleware(logger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/crash", nil)

	require.NotPanics(t, func() {
		mw(next).ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")
}

func TestRecoveryMiddleware_PanicWithNonError(t *testing.T) {
	var buf nopWriter
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went very wrong")
	})

	mw := RecoveryMiddleware(logger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bad", nil)

	require.NotPanics(t, func() {
		mw(next).ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

func TestRecoveryMiddleware_LogsRequestID(t *testing.T) {
	var buf safeBuffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("oops")
	})

	mw := RecoveryMiddleware(logger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	// Inject a request ID into the context so we can verify it is logged.
	ctx := context.WithValue(req.Context(), requestIDKey, "req-abc-123")
	req = req.WithContext(ctx)

	require.NotPanics(t, func() {
		mw(next).ServeHTTP(rec, req)
	})

	assert.Contains(t, buf.String(), "req-abc-123")
}

func TestGetRequestID_Empty(t *testing.T) {
	id := GetRequestID(context.Background())
	assert.Equal(t, "", id)
}

func TestGetRequestID_Set(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, "test-id-42")
	assert.Equal(t, "test-id-42", GetRequestID(ctx))
}

func TestGetRequestID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, 999)
	assert.Equal(t, "", GetRequestID(ctx))
}

// responseWriter tests

func TestResponseWriter_WriteHeader_OnlyOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, status: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)
	rw.WriteHeader(http.StatusAccepted) // should be ignored

	assert.Equal(t, http.StatusCreated, rw.status)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestResponseWriter_Write_SetsStatusOK(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, status: http.StatusOK}

	n, err := rw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.True(t, rw.wroteHeader)
	assert.Equal(t, http.StatusOK, rw.status)
}

// helpers

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

type safeBuffer struct {
	data []byte
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *safeBuffer) String() string { return string(b.data) }
