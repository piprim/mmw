package gateway_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/outbound/gateway"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

func TestHTTP[[.Entity | pascal]]Gateway(t *testing.T) {
	item := [[.Entity]].New([[.Entity]].ReconstituteLabel("Bonjour"))

	t.Run("posts the label as JSON with the bearer token", func(t *testing.T) {
		var gotAuth, gotBody string
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			w.WriteHeader(http.StatusAccepted)
		}))
		defer provider.Close()

		err := gateway.NewHTTP[[.Entity | pascal]]Gateway(provider.URL, "secret").[[.Entity | pascal]]Send(context.Background(), item)
		if err != nil {
			t.Fatalf("Send error: %v", err)
		}
		if gotAuth != "Bearer secret" || gotBody != `{"label":"Bonjour"}` {
			t.Errorf("auth/body = %q/%q", gotAuth, gotBody)
		}
	})

	t.Run("a non-2xx answer is ErrGatewayUnavailable", func(t *testing.T) {
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer provider.Close()

		err := gateway.NewHTTP[[.Entity | pascal]]Gateway(provider.URL, "").[[.Entity | pascal]]Send(context.Background(), item)
		if !errors.Is(err, [[.Entity]].ErrGatewayUnavailable) {
			t.Errorf("error = %v, want ErrGatewayUnavailable", err)
		}
	})

	t.Run("an unreachable provider is ErrGatewayUnavailable", func(t *testing.T) {
		err := gateway.NewHTTP[[.Entity | pascal]]Gateway("http://127.0.0.1:1", "").[[.Entity | pascal]]Send(context.Background(), item)
		if !errors.Is(err, [[.Entity]].ErrGatewayUnavailable) {
			t.Errorf("error = %v, want ErrGatewayUnavailable", err)
		}
	})

	t.Run("Health reports a 5xx answer as ErrGatewayUnavailable", func(t *testing.T) {
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer provider.Close()

		_, err := gateway.NewHTTP[[.Entity | pascal]]Gateway(provider.URL, "").Health(context.Background())
		if !errors.Is(err, [[.Entity]].ErrGatewayUnavailable) {
			t.Errorf("error = %v, want ErrGatewayUnavailable", err)
		}
	})

	t.Run("Health returns the status code", func(t *testing.T) {
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer provider.Close()

		status, err := gateway.NewHTTP[[.Entity | pascal]]Gateway(provider.URL, "").Health(context.Background())
		if err != nil || status != http.StatusOK {
			t.Errorf("status/err = %v/%v", status, err)
		}
	})
}
