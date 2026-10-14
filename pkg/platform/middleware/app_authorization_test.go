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

	pfauthctx "github.com/piprim/mmw/pkg/platform/authctx"
	. "github.com/piprim/mmw/pkg/platform/middleware"
)

func TestAppAuthorizationMiddleware(t *testing.T) {
	userID := uuid.New()

	serve := func(checker AccessChecker, req *http.Request) (called bool, status int, body string) {
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		rr := httptest.NewRecorder()
		AppAuthorizationMiddleware("cpro", checker, slog.Default())(next).ServeHTTP(rr, req)

		return called, rr.Code, rr.Body.String()
	}
	authenticated := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/", nil)

		return req.WithContext(pfauthctx.WithUserID(req.Context(), userID))
	}
	// checkerFor answers allowed only for userID and the "cpro" application.
	checkerFor := func(allowed bool, err error) AccessChecker {
		return func(_ context.Context, uid uuid.UUID, application string) (bool, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, "cpro", application)

			return allowed, err
		}
	}

	t.Run("a user who may access the application goes through", func(t *testing.T) {
		called, status, _ := serve(checkerFor(true, nil), authenticated())

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("a user who may not access the application gets 403", func(t *testing.T) {
		called, status, body := serve(checkerFor(false, nil), authenticated())

		assert.False(t, called)
		assert.Equal(t, http.StatusForbidden, status)
		assert.JSONEq(t, `{"code":"permission_denied","message":"access to this application is not allowed"}`, body)
	})

	t.Run("a checker failure denies access with 503, never 403 nor a pass", func(t *testing.T) {
		called, status, body := serve(checkerFor(true, errors.New("db down")), authenticated())

		assert.False(t, called)
		assert.Equal(t, http.StatusServiceUnavailable, status)
		assert.JSONEq(t, `{"code":"unavailable","message":"cannot check access to this application"}`, body)
	})

	t.Run("a request that was not authenticated gets 401 without asking the checker", func(t *testing.T) {
		checker := func(context.Context, uuid.UUID, string) (bool, error) {
			t.Error("the checker must not be called without an authenticated user")

			return true, nil
		}
		called, status, _ := serve(checker, httptest.NewRequest(http.MethodPost, "/", nil))

		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, status)
	})
}
