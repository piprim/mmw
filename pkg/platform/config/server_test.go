package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer_SetDefaults_ZeroValues(t *testing.T) {
	s := &Server{}
	s.SetDefaults()

	assert.Equal(t, readHeaderTimeout, s.ReadHeaderTimeout)
	assert.Equal(t, idleTimeout, s.IdleTimeout)
	assert.Equal(t, shutdownTimeout, s.ShutdownTimeout)
}

func TestServer_SetDefaults_PreservesNonZero(t *testing.T) {
	s := &Server{
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   15 * time.Second,
	}
	s.SetDefaults()

	assert.Equal(t, 10*time.Second, s.ReadHeaderTimeout)
	assert.Equal(t, 60*time.Second, s.IdleTimeout)
	assert.Equal(t, 15*time.Second, s.ShutdownTimeout)
}

func TestServer_SetDefaults_PartialZero(t *testing.T) {
	s := &Server{
		ReadHeaderTimeout: 10 * time.Second,
		// IdleTimeout and ShutdownTimeout are zero
	}
	s.SetDefaults()

	assert.Equal(t, 10*time.Second, s.ReadHeaderTimeout)
	assert.Equal(t, idleTimeout, s.IdleTimeout)
	assert.Equal(t, shutdownTimeout, s.ShutdownTimeout)
}

func TestServer_URL_Basic(t *testing.T) {
	s := &Server{
		Scheme: "http",
		Host:   "localhost",
		Port:   8080,
	}
	url := s.URL("/api/users", nil)
	assert.Equal(t, "http://localhost:8080/api/users", url)
}

func TestServer_URL_WithQueries(t *testing.T) {
	s := &Server{
		Scheme: "https",
		Host:   "api.example.com",
		Port:   443,
	}
	url := s.URL("/search", map[string]string{"q": "test"})
	assert.Contains(t, url, "q=test")
	assert.Contains(t, url, "/search")
}

func TestServer_URL_StandardHTTPPort(t *testing.T) {
	s := &Server{
		Scheme: "http",
		Host:   "localhost",
		Port:   80,
	}
	url := s.URL("/", nil)
	// Port 80 on http is omitted
	assert.NotContains(t, url, ":80")
}

func TestServer_URL_StandardHTTPSPort(t *testing.T) {
	s := &Server{
		Scheme: "https",
		Host:   "example.com",
		Port:   443,
	}
	url := s.URL("/", nil)
	// Port 443 on https is omitted
	assert.NotContains(t, url, ":443")
}

func TestServer_URL_NonStandardPortOnHTTP(t *testing.T) {
	s := &Server{
		Scheme: "http",
		Host:   "localhost",
		Port:   9090,
	}
	url := s.URL("/health", nil)
	assert.Contains(t, url, ":9090")
}

func TestServer_URL_EmptyPath(t *testing.T) {
	s := &Server{
		Scheme: "http",
		Host:   "localhost",
		Port:   3000,
	}
	url := s.URL("", nil)
	assert.Equal(t, "http://localhost:3000", url)
}
