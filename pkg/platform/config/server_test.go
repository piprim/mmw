package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer_SetDefaults(t *testing.T) {
	t.Run("sets all defaults when fields are zero", func(t *testing.T) {
		s := &Server{}
		s.SetDefaults()

		assert.Equal(t, readHeaderTimeout, s.ReadHeaderTimeout)
		assert.Equal(t, idleTimeout, s.IdleTimeout)
		assert.Equal(t, shutdownTimeout, s.ShutdownTimeout)
	})

	t.Run("preserves non-zero values", func(t *testing.T) {
		s := &Server{
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   15 * time.Second,
		}
		s.SetDefaults()

		assert.Equal(t, 10*time.Second, s.ReadHeaderTimeout)
		assert.Equal(t, 60*time.Second, s.IdleTimeout)
		assert.Equal(t, 15*time.Second, s.ShutdownTimeout)
	})

	t.Run("fills only zero fields when partially set", func(t *testing.T) {
		s := &Server{ReadHeaderTimeout: 10 * time.Second}
		s.SetDefaults()

		assert.Equal(t, 10*time.Second, s.ReadHeaderTimeout)
		assert.Equal(t, idleTimeout, s.IdleTimeout)
		assert.Equal(t, shutdownTimeout, s.ShutdownTimeout)
	})
}

func TestServer_DebugEnabled(t *testing.T) {
	t.Run("defaults to false", func(t *testing.T) {
		s := &Server{}
		assert.False(t, s.DebugEnabled)
	})
}

func TestServer_URL(t *testing.T) {
	t.Run("builds basic URL", func(t *testing.T) {
		s := &Server{Scheme: "http", Host: "localhost", Port: 8080}
		assert.Equal(t, "http://localhost:8080/api/users", s.URL("/api/users", nil))
	})

	t.Run("appends query parameters", func(t *testing.T) {
		s := &Server{Scheme: "https", Host: "api.example.com", Port: 443}
		url := s.URL("/search", map[string]string{"q": "test"})
		assert.Contains(t, url, "q=test")
		assert.Contains(t, url, "/search")
	})

	t.Run("omits standard HTTP port 80", func(t *testing.T) {
		s := &Server{Scheme: "http", Host: "localhost", Port: 80}
		assert.NotContains(t, s.URL("/", nil), ":80")
	})

	t.Run("omits standard HTTPS port 443", func(t *testing.T) {
		s := &Server{Scheme: "https", Host: "example.com", Port: 443}
		assert.NotContains(t, s.URL("/", nil), ":443")
	})

	t.Run("includes non-standard port on HTTP", func(t *testing.T) {
		s := &Server{Scheme: "http", Host: "localhost", Port: 9090}
		assert.Contains(t, s.URL("/health", nil), ":9090")
	})

	t.Run("handles empty path", func(t *testing.T) {
		s := &Server{Scheme: "http", Host: "localhost", Port: 3000}
		assert.Equal(t, "http://localhost:3000", s.URL("", nil))
	})
}
