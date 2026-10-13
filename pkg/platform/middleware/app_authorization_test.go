package middleware_test

import (
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
	serve := func(ctx func() *http.Request) (called bool, status int, body string) {
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		rr := httptest.NewRecorder()
		AppAuthorizationMiddleware("cpro", slog.Default())(next).ServeHTTP(rr, ctx())

		return called, rr.Code, rr.Body.String()
	}
	authenticated := func(apps ...string) func() *http.Request {
		return func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			ctx := pfauthctx.WithUserID(req.Context(), uuid.New())

			return req.WithContext(pfauthctx.WithApplications(ctx, apps))
		}
	}

	t.Run("a user who may access the application goes through", func(t *testing.T) {
		called, status, _ := serve(authenticated("other", "cpro"))

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("a user who may not access the application gets 403", func(t *testing.T) {
		called, status, body := serve(authenticated("other"))

		assert.False(t, called)
		assert.Equal(t, http.StatusForbidden, status)
		assert.JSONEq(t, `{"code":"permission_denied","message":"access to this application is not allowed"}`, body)
	})

	t.Run("a user without any application gets 403", func(t *testing.T) {
		called, status, _ := serve(authenticated())

		assert.False(t, called)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("a request that was not authenticated gets 401", func(t *testing.T) {
		called, status, _ := serve(func() *http.Request { return httptest.NewRequest(http.MethodPost, "/", nil) })

		assert.False(t, called)
		assert.Equal(t, http.StatusUnauthorized, status)
	})
}
