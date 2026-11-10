package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// ErrInvalidAllowedOrigin is returned when an allowed origin is not a bare
// origin (scheme://host[:port]) or is the "*" wildcard.
var ErrInvalidAllowedOrigin = errors.New("invalid allowed origin")

const (
	readHeaderTimeout = 5 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 30 * time.Second
)

// Server defines everything the platform HTTPServer needs to know.
type Server struct {
	Scheme            string        `mapstructure:"scheme"`
	Host              string        `mapstructure:"host"`
	Port              Port          `mapstructure:"port"`
	ReadHeaderTimeout time.Duration `mapstructure:"read-header-timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle-timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown-timeout"`
	// For CORS
	AllowedOrigins []string `mapstructure:"allowed-origins"`
	// DebugEnabled exposes pprof routes and gRPC reflection.
	// Controlled via SERVER_DEBUG_ENABLED env var or debug-enabled in TOML.
	DebugEnabled bool `env:"SERVER_DEBUG_ENABLED" mapstructure:"debug-enabled"`
}

// SetDefaults ensures the server won't crash if the TOML is missing fields.
func (s *Server) SetDefaults() {
	if s.ReadHeaderTimeout == 0 {
		s.ReadHeaderTimeout = readHeaderTimeout
	}
	if s.IdleTimeout == 0 {
		s.IdleTimeout = idleTimeout
	}
	if s.ShutdownTimeout == 0 {
		s.ShutdownTimeout = shutdownTimeout
	}
}

// Validate rejects allowed origins the platform cannot enforce: the CORS and
// origin checks send credentials, so each origin must be explicit, bare
// (scheme://host[:port], no path or trailing slash) and never "*".
func (s *Server) Validate() error {
	for _, origin := range s.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
			u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
			return fmt.Errorf("%w: %q must look like https://app.example.com", ErrInvalidAllowedOrigin, origin)
		}
	}

	return nil
}

func (s *Server) URL(path string, queries map[string]string) string {
	port := ""
	if (s.Scheme != "http" || s.Port != 80) && (s.Scheme != "https" || s.Port != 443) {
		port = s.Port.String()
	}

	u := &url.URL{
		Scheme: s.Scheme,
		Host:   s.Host + port,
		Path:   path,
	}

	q := u.Query()
	for key, value := range queries {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	return u.String()
}
