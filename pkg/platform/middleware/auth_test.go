package middleware_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pfauthctx "github.com/piprim/mmw/pkg/platform/authctx"
	. "github.com/piprim/mmw/pkg/platform/middleware"
	"github.com/piprim/mmw/pkg/platform/sessioncookie"
)

func TestBearerAuthMiddleware(t *testing.T) {
	t.Run("valid token injects user ID into context", func(t *testing.T) {
		userID := uuid.New()
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			return TokenValidation{UserID: userID}, nil
		}

		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			id, err := pfauthctx.UserIDFromContext(r.Context())
			require.NoError(t, err)
			assert.Equal(t, userID, id)
			w.WriteHeader(http.StatusOK)
		})

		mw := BearerAuthMiddleware(validate, slog.Default(), nil)
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("renewed token is sent back in the response header", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			return TokenValidation{UserID: uuid.New(), RenewedToken: "new-token"}, nil
		}
		// The handler fails: the header must be there anyway.
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})

		mw := BearerAuthMiddleware(validate, slog.Default(), nil)
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer expired-token")
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)

		assert.Equal(t, "new-token", rr.Header().Get(RenewedTokenHeader))
	})

	t.Run("no renewal sets no header", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			return TokenValidation{UserID: uuid.New()}, nil
		}
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

		mw := BearerAuthMiddleware(validate, slog.Default(), nil)
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer token")
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)

		assert.Empty(t, rr.Header().Values(RenewedTokenHeader))
	})

	t.Run("missing token returns 401", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			return TokenValidation{UserID: uuid.New()}, nil
		}
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		mw := BearerAuthMiddleware(validate, slog.Default(), nil)
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			return TokenValidation{}, errors.New("invalid token")
		}
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		mw := BearerAuthMiddleware(validate, slog.Default(), nil)
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer bad-token")
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("excluded path skips auth", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			return TokenValidation{}, errors.New("should not be called")
		}
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		mw := BearerAuthMiddleware(validate, slog.Default(), []string{"/public/"})
		req := httptest.NewRequest(http.MethodGet, "/public/health", nil)
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestAuthMiddleware_SessionCookie(t *testing.T) {
	cookie, err := sessioncookie.New(sessioncookie.Config{})
	require.NoError(t, err)

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("authenticates from the session cookie alone", func(t *testing.T) {
		var seen string
		validate := func(_ context.Context, token string) (TokenValidation, error) {
			seen = token
			return TokenValidation{UserID: uuid.New()}, nil
		}

		mw := AuthMiddleware(validate, slog.Default(), nil, WithSessionCookie(cookie))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.AddCookie(&http.Cookie{Name: "__Secure-session", Value: "from-cookie"})
		rr := httptest.NewRecorder()
		mw(ok).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "from-cookie", seen)
	})

	t.Run("the cookie wins over the Authorization header", func(t *testing.T) {
		var seen string
		validate := func(_ context.Context, token string) (TokenValidation, error) {
			seen = token
			return TokenValidation{UserID: uuid.New()}, nil
		}

		mw := AuthMiddleware(validate, slog.Default(), nil, WithSessionCookie(cookie))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.AddCookie(&http.Cookie{Name: "__Secure-session", Value: "from-cookie"})
		req.Header.Set("Authorization", "Bearer from-header")
		mw(ok).ServeHTTP(httptest.NewRecorder(), req)

		assert.Equal(t, "from-cookie", seen)
	})

	t.Run("falls back to the Authorization header without a cookie", func(t *testing.T) {
		var seen string
		validate := func(_ context.Context, token string) (TokenValidation, error) {
			seen = token
			return TokenValidation{UserID: uuid.New()}, nil
		}

		mw := AuthMiddleware(validate, slog.Default(), nil, WithSessionCookie(cookie))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer from-header")
		rr := httptest.NewRecorder()
		mw(ok).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "from-header", seen)
	})

	t.Run("without cookie or header the request is refused", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			t.Fatal("validate must not be called")
			return TokenValidation{}, nil
		}

		mw := AuthMiddleware(validate, slog.Default(), nil, WithSessionCookie(cookie))
		rr := httptest.NewRecorder()
		mw(ok).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", nil))

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("a token renewed for a cookie request is set back in the cookie", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			return TokenValidation{UserID: uuid.New(), RenewedToken: "new-token"}, nil
		}
		// The handler fails: the cookie must be there anyway.
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })

		mw := AuthMiddleware(validate, slog.Default(), nil, WithSessionCookie(cookie))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.AddCookie(&http.Cookie{Name: "__Secure-session", Value: "expired"})
		rr := httptest.NewRecorder()
		mw(next).ServeHTTP(rr, req)

		cookies := rr.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "__Secure-session", cookies[0].Name)
		assert.Equal(t, "new-token", cookies[0].Value)
		assert.Equal(t, "new-token", rr.Header().Get(RenewedTokenHeader), "legacy clients still get the header")
	})

	t.Run("a token renewed for a Bearer request sets no cookie", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			return TokenValidation{UserID: uuid.New(), RenewedToken: "new-token"}, nil
		}

		mw := AuthMiddleware(validate, slog.Default(), nil, WithSessionCookie(cookie))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer expired")
		rr := httptest.NewRecorder()
		mw(ok).ServeHTTP(rr, req)

		assert.Empty(t, rr.Result().Cookies())
		assert.Equal(t, "new-token", rr.Header().Get(RenewedTokenHeader))
	})

	t.Run("without the option the cookie is ignored", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (TokenValidation, error) {
			t.Fatal("validate must not be called")
			return TokenValidation{}, nil
		}

		mw := AuthMiddleware(validate, slog.Default(), nil)
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.AddCookie(&http.Cookie{Name: "__Secure-session", Value: "from-cookie"})
		rr := httptest.NewRecorder()
		mw(ok).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}
