package oglpfconfig

import (
	"net/url"
	"time"
)

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
