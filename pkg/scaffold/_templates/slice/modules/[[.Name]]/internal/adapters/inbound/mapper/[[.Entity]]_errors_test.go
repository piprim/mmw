package mapper_test

import (
	"errors"
	"fmt"
	"testing"

	"[[.PlatformPath]]/pkg/platform"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"

	"[[.ModulePath]]/internal/adapters/inbound/mapper"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

func TestDomainErrorFor_[[.Entity | pascal]]Sentinels(t *testing.T) {
	cases := []struct {
		name     string
		input    error
		wantCode platform.ErrorCode
	}{
[[- if ne .Shape "gateway"]]
		{"[[.Entity]].ErrNotFound", [[.Entity]].ErrNotFound, platform.ErrorCode([[.Name]]def.ErrorCodeNotFound)},
		{"[[.Entity]].ErrInvalidID", [[.Entity]].ErrInvalidID, platform.ErrorCode([[.Name]]def.ErrorCodeInvalidID)},
[[- else]]
		{"[[.Entity]].ErrGatewayUnavailable", [[.Entity]].ErrGatewayUnavailable, platform.ErrorCode([[.Name]]def.ErrorCodeGatewayUnavailable)},
[[- end]]
		{"[[.Entity]].ErrInvalidLabel", [[.Entity]].ErrInvalidLabel, platform.ErrorCode([[.Name]]def.ErrorCodeInvalidLabel)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			domErr, ok := errors.AsType[*platform.DomainError](mapper.DomainErrorFor(tc.input))
			if !ok {
				t.Fatalf("expected *platform.DomainError")
			}
			if domErr.Code != tc.wantCode || domErr.Message == "" {
				t.Errorf("Code/Message = %v/%q, want %v", domErr.Code, domErr.Message, tc.wantCode)
			}
		})
	}

	t.Run("unwraps a wrapped sentinel", func(t *testing.T) {
		_, ok := errors.AsType[*platform.DomainError](mapper.DomainErrorFor(fmt.Errorf("context: %w", [[.Entity]].ErrInvalidLabel)))
		if !ok {
			t.Fatal("expected *platform.DomainError for wrapped domain error")
		}
	})
}
