package sessioncookie_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piprim/mmw/pkg/platform/sessioncookie"
)

func newCookie(t *testing.T, cfg sessioncookie.Config) *sessioncookie.Cookie {
	t.Helper()

	c, err := sessioncookie.New(cfg)
	require.NoError(t, err)

	return c
}

func setCookieOf(t *testing.T, h http.Header) *http.Cookie {
	t.Helper()

	resp := http.Response{Header: h}
	cookies := resp.Cookies()
	require.Len(t, cookies, 1)

	return cookies[0]
}

func TestNew(t *testing.T) {
	t.Run("zero config takes the secure defaults", func(t *testing.T) {
		c := newCookie(t, sessioncookie.Config{})

		h := http.Header{}
		c.Set(h, "tok")

		got := setCookieOf(t, h)
		assert.Equal(t, "__Secure-session", got.Name)
		assert.Equal(t, 24*60*60, got.MaxAge)
	})

	t.Run("a name without the __Secure- prefix is refused", func(t *testing.T) {
		_, err := sessioncookie.New(sessioncookie.Config{Name: "session"})

		require.ErrorIs(t, err, sessioncookie.ErrInvalidName)
	})

	t.Run("a negative max age is refused", func(t *testing.T) {
		_, err := sessioncookie.New(sessioncookie.Config{MaxAge: -time.Second})

		require.ErrorIs(t, err, sessioncookie.ErrInvalidMaxAge)
	})
}

func TestSet(t *testing.T) {
	t.Run("sets every attribute the session cookie must carry", func(t *testing.T) {
		c := newCookie(t, sessioncookie.Config{
			Name: "__Secure-cpro", Domain: "example.com", MaxAge: 2 * time.Hour,
		})

		h := http.Header{}
		c.Set(h, "tok")

		got := setCookieOf(t, h)
		assert.Equal(t, "__Secure-cpro", got.Name)
		assert.Equal(t, "tok", got.Value)
		assert.Equal(t, "example.com", got.Domain)
		assert.Equal(t, "/", got.Path)
		assert.Equal(t, 7200, got.MaxAge)
		assert.True(t, got.HttpOnly, "HttpOnly keeps the token away from scripts")
		assert.True(t, got.Secure, "the __Secure- prefix requires Secure")
		assert.Equal(t, http.SameSiteLaxMode, got.SameSite)
	})

	t.Run("no domain gives a host-only cookie", func(t *testing.T) {
		c := newCookie(t, sessioncookie.Config{})

		h := http.Header{}
		c.Set(h, "tok")

		assert.Empty(t, setCookieOf(t, h).Domain)
	})
}

func TestClear(t *testing.T) {
	t.Run("expires the cookie with the same name, domain and path", func(t *testing.T) {
		c := newCookie(t, sessioncookie.Config{Domain: "example.com"})

		h := http.Header{}
		c.Clear(h)

		got := setCookieOf(t, h)
		assert.Equal(t, "__Secure-session", got.Name)
		assert.Equal(t, "example.com", got.Domain)
		assert.Equal(t, "/", got.Path)
		assert.Empty(t, got.Value)
		assert.Negative(t, got.MaxAge)
		assert.True(t, got.HttpOnly)
		assert.True(t, got.Secure)
	})
}

func TestToken(t *testing.T) {
	c := newCookie(t, sessioncookie.Config{})

	t.Run("reads the token from the Cookie header", func(t *testing.T) {
		h := http.Header{"Cookie": {"other=1; __Secure-session=tok"}}

		assert.Equal(t, "tok", c.Token(h))
	})

	t.Run("returns empty when the cookie is absent", func(t *testing.T) {
		assert.Empty(t, c.Token(http.Header{"Cookie": {"other=1"}}))
		assert.Empty(t, c.Token(http.Header{}))
	})
}
