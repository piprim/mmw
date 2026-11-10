package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	pfconfig "github.com/piprim/mmw/pkg/platform/config"
	"github.com/piprim/mmw/pkg/platform/server"
)

// newHandler builds the assembled server around an application that records whether it was reached.
func newHandler(allowed []string) (h http.Handler, reached *bool) {
	reached = new(bool)
	app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})

	srv := server.NewHTTPServer(server.HTTPServerInfra{
		Config:  &pfconfig.Server{Host: "localhost", AllowedOrigins: allowed},
		Handler: app,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	return srv.Handler(), reached
}

func TestHTTPServer_originGuard(t *testing.T) {
	allowed := []string{"https://app.example.com"}

	t.Run("refuses a cross-site POST before it reaches the application", func(t *testing.T) {
		h, reached := newHandler(allowed)
		req := httptest.NewRequest(http.MethodPost, "/core.v1.TodoService/CreateTodo", nil)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("Origin", "https://evil.example.com")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.False(t, *reached)
	})

	t.Run("lets a POST from an allowed origin through", func(t *testing.T) {
		h, reached := newHandler(allowed)
		req := httptest.NewRequest(http.MethodPost, "/core.v1.TodoService/CreateTodo", nil)
		req.Header.Set("Sec-Fetch-Site", "same-site")
		req.Header.Set("Origin", "https://app.example.com")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, *reached)
	})

	t.Run("answers a preflight from an allowed origin with credentials allowed", func(t *testing.T) {
		h, reached := newHandler(allowed)
		req := httptest.NewRequest(http.MethodOptions, "/core.v1.TodoService/CreateTodo", nil)
		req.Header.Set("Origin", "https://app.example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		assert.False(t, *reached, "the preflight is answered by the CORS layer")
	})
}
