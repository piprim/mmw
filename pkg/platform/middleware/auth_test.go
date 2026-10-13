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
)

func TestBearerAuthMiddleware(t *testing.T) {
	t.Run("valid token injects user ID into context", func(t *testing.T) {
		userID := uuid.New()
		validate := func(_ context.Context, _ string) (Identity, error) {
			return Identity{UserID: userID}, nil
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

	t.Run("valid token injects the user's applications into context", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (Identity, error) {
			return Identity{UserID: uuid.New(), Applications: []string{"cpro", "other"}}, nil
		}

		var got []string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = pfauthctx.ApplicationsFromContext(r.Context())
		})

		mw := BearerAuthMiddleware(validate, slog.Default(), nil)
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		mw(next).ServeHTTP(httptest.NewRecorder(), req)

		assert.Equal(t, []string{"cpro", "other"}, got)
	})

	t.Run("missing token returns 401", func(t *testing.T) {
		validate := func(_ context.Context, _ string) (Identity, error) {
			return Identity{UserID: uuid.New()}, nil
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
		validate := func(_ context.Context, _ string) (Identity, error) {
			return Identity{}, errors.New("invalid token")
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
		validate := func(_ context.Context, _ string) (Identity, error) {
			return Identity{}, errors.New("should not be called")
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
