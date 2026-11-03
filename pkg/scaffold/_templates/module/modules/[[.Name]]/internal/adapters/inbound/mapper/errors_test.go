package mapper_test

import (
	"errors"
	"testing"

	"[[.ModulePath]]/internal/adapters/inbound/mapper"
)

func TestDomainErrorFor_Other(t *testing.T) {
	t.Run("passes a non-domain error through", func(t *testing.T) {
		infra := errors.New("db connection refused")
		if got := mapper.DomainErrorFor(infra); got != infra {
			t.Errorf("expected original error, got %v", got)
		}
	})
}
