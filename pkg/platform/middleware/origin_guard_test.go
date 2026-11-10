package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/piprim/mmw/pkg/platform/middleware"
)

// serveThroughGuard runs a request through OriginGuard and reports whether the handler behind it was reached.
func serveThroughGuard(t *testing.T, allowed []string, method string, headers map[string]string) (status int, reached bool) {
	t.Helper()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(method, "/core.v1.TodoService/CreateTodo", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rr := httptest.NewRecorder()
	OriginGuard(allowed, slog.Default())(next).ServeHTTP(rr, req)

	return rr.Code, reached
}

func TestOriginGuard_crossSite(t *testing.T) {
	t.Run("a cross-site POST is refused even from an allowed origin", func(t *testing.T) {
		status, reached := serveThroughGuard(t, []string{"https://app.example.com"}, http.MethodPost, map[string]string{
			"Sec-Fetch-Site": "cross-site",
			"Origin":         "https://app.example.com",
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.False(t, reached)
	})
}

func TestOriginGuard_allowList(t *testing.T) {
	allowed := []string{"https://app.example.com", "https://admin.example.com"}

	t.Run("a same-site POST from an unknown origin is refused", func(t *testing.T) {
		// A sibling subdomain is same-site: only the allow-list tells it apart.
		status, reached := serveThroughGuard(t, allowed, http.MethodPost, map[string]string{
			"Sec-Fetch-Site": "same-site",
			"Origin":         "https://evil.example.com",
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.False(t, reached)
	})

	t.Run("a POST from an allowed origin passes", func(t *testing.T) {
		status, reached := serveThroughGuard(t, allowed, http.MethodPost, map[string]string{
			"Sec-Fetch-Site": "same-site",
			"Origin":         "https://admin.example.com",
		})

		assert.Equal(t, http.StatusOK, status)
		assert.True(t, reached)
	})

	t.Run("an Origin differing by scheme, port or suffix is not allowed", func(t *testing.T) {
		for _, origin := range []string{
			"http://app.example.com", "https://app.example.com:8443", "https://app.example.com.evil.com",
			"HTTPS://APP.EXAMPLE.COM",
		} {
			status, _ := serveThroughGuard(t, allowed, http.MethodPost, map[string]string{"Origin": origin})

			assert.Equal(t, http.StatusForbidden, status, origin)
		}
	})
}

func TestOriginGuard_whatItLetsThrough(t *testing.T) {
	allowed := []string{"https://app.example.com"}

	t.Run("safe methods are not checked", func(t *testing.T) {
		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
			status, reached := serveThroughGuard(t, allowed, method, map[string]string{
				"Sec-Fetch-Site": "cross-site",
				"Origin":         "https://evil.example.com",
			})

			assert.Equal(t, http.StatusOK, status, method)
			assert.True(t, reached, method)
		}
	})

	t.Run("a non-browser client sending neither header is not blocked", func(t *testing.T) {
		status, reached := serveThroughGuard(t, allowed, http.MethodPost, nil)

		assert.Equal(t, http.StatusOK, status)
		assert.True(t, reached)
	})

	t.Run("a same-origin POST passes without being in the allow-list", func(t *testing.T) {
		status, reached := serveThroughGuard(t, nil, http.MethodPost, map[string]string{
			"Sec-Fetch-Site": "same-origin",
			"Origin":         "http://localhost:4200",
		})

		assert.Equal(t, http.StatusOK, status)
		assert.True(t, reached)
	})

	t.Run("every state-changing method is checked", func(t *testing.T) {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			status, _ := serveThroughGuard(t, allowed, method, map[string]string{"Origin": "https://evil.example.com"})

			assert.Equal(t, http.StatusForbidden, status, method)
		}
	})
}

func TestOriginGuard_headerCombinations(t *testing.T) {
	allowed := []string{"https://app.example.com"}

	cases := []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{"cross-site without Origin", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"same-origin with a foreign Origin", map[string]string{
			"Sec-Fetch-Site": "same-origin", "Origin": "https://evil.example.com",
		}, http.StatusOK},
		{"user-initiated (none) without Origin", map[string]string{"Sec-Fetch-Site": "none"}, http.StatusOK},
		{"user-initiated (none) with an allowed Origin", map[string]string{
			"Sec-Fetch-Site": "none", "Origin": "https://app.example.com",
		}, http.StatusOK},
		{"opaque Origin (sandboxed frame)", map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "null"}, http.StatusForbidden},
		{"opaque Origin without Sec-Fetch-Site", map[string]string{"Origin": "null"}, http.StatusForbidden},
		{"unknown Sec-Fetch-Site value falls back to the Origin check", map[string]string{
			"Sec-Fetch-Site": "bogus", "Origin": "https://evil.example.com",
		}, http.StatusForbidden},
		{"unknown Sec-Fetch-Site value with an allowed Origin", map[string]string{
			"Sec-Fetch-Site": "bogus", "Origin": "https://app.example.com",
		}, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, _ := serveThroughGuard(t, allowed, http.MethodPost, tc.headers)

			assert.Equal(t, tc.want, status)
		})
	}
}

func TestOriginGuard_refusalBody(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()

	OriginGuard([]string{"https://app.example.com"}, slog.Default())(next).ServeHTTP(rr, req)

	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"code":"permission_denied","message":"request origin not allowed"}`, rr.Body.String())
}
