// Package sessioncookie carries the access token of a browser session in an
// HttpOnly cookie, so scripts cannot read it.
//
// The cookie is shared by the applications of a domain when Config.Domain is
// set. It is read by the authentication middleware and written by the login,
// logout and renewal paths, so both sides must use the same Config.
package sessioncookie

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	// secureNamePrefix makes the browser refuse the cookie unless it was set
	// over HTTPS with the Secure attribute.
	secureNamePrefix = "__Secure-"
	defaultName      = secureNamePrefix + "session"
	// defaultMaxAge must cover the longest session: the renewal needs the
	// expired access token to still reach the server.
	defaultMaxAge = 24 * time.Hour
)

var (
	// ErrInvalidName is returned when the cookie name lacks the __Secure- prefix.
	ErrInvalidName = errors.New("session cookie name must start with " + secureNamePrefix)
	// ErrInvalidMaxAge is returned when the max age is negative.
	ErrInvalidMaxAge = errors.New("session cookie max age must not be negative")
)

// Config is the session cookie setting. The zero value takes the defaults.
type Config struct {
	// Name of the cookie; must start with "__Secure-". Default "__Secure-session".
	Name string `mapstructure:"name"`
	// Domain shares the cookie with every subdomain. Empty means host-only.
	Domain string `mapstructure:"domain"`
	// MaxAge is how long the browser keeps the cookie. It must not be shorter
	// than the longest session. Zero means 24h.
	MaxAge time.Duration `mapstructure:"max-age"`
}

// Cookie reads, sets and clears the session cookie. Build it with New.
type Cookie struct {
	name   string
	domain string
	maxAge int
}

// New validates the config, applies the defaults and returns the Cookie.
func New(cfg Config) (*Cookie, error) {
	if cfg.Name == "" {
		cfg.Name = defaultName
	}

	if !strings.HasPrefix(cfg.Name, secureNamePrefix) {
		return nil, ErrInvalidName
	}

	if cfg.MaxAge < 0 {
		return nil, ErrInvalidMaxAge
	}

	if cfg.MaxAge == 0 {
		cfg.MaxAge = defaultMaxAge
	}

	return &Cookie{name: cfg.Name, domain: cfg.Domain, maxAge: int(cfg.MaxAge.Seconds())}, nil
}

// Token returns the access token carried by the request headers, or "" when absent.
func (c *Cookie) Token(h http.Header) string {
	r := http.Request{Header: h}

	cookie, err := r.Cookie(c.name)
	if err != nil {
		return ""
	}

	return cookie.Value
}

// Set adds a Set-Cookie header carrying the token to h.
func (c *Cookie) Set(h http.Header, token string) {
	h.Add("Set-Cookie", c.build(token, c.maxAge).String())
}

// Clear adds a Set-Cookie header that makes the browser drop the cookie.
func (c *Cookie) Clear(h http.Header) {
	h.Add("Set-Cookie", c.build("", -1).String())
}

func (c *Cookie) build(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     c.name,
		Value:    value,
		Domain:   c.domain,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		// Lax, not Strict: Strict would drop the cookie on the first navigation
		// from an external link. Cross-site POSTs stay blocked either way.
		SameSite: http.SameSiteLaxMode,
	}
}
