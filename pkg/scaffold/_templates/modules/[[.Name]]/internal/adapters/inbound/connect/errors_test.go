package connect

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"[[.PlatformPath]]/pkg/platform"
	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
)

func TestConnectErrorFrom(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode connect.Code
	}{
[[- if .Database]]
		{"NotFound", &platform.DomainError{Code: platform.ErrorCode([[.Name]]def.ErrorCodeNotFound), Message: "x"}, connect.CodeNotFound},
		{"InvalidID", &platform.DomainError{Code: platform.ErrorCode([[.Name]]def.ErrorCodeInvalidID), Message: "x"}, connect.CodeInvalidArgument},
[[- else]]
		{"GatewayUnavailable", &platform.DomainError{Code: platform.ErrorCode([[.Name]]def.ErrorCodeGatewayUnavailable), Message: "x"}, connect.CodeUnavailable},
[[- end]]
		{"InvalidLabel", &platform.DomainError{Code: platform.ErrorCode([[.Name]]def.ErrorCodeInvalidLabel), Message: "x"}, connect.CodeInvalidArgument},
		{"unknown domain code is internal", &platform.DomainError{Code: 9999, Message: "x"}, connect.CodeInternal},
		{"non-domain error is internal", errors.New("db connection refused"), connect.CodeInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := connectErrorFrom(tc.err).Code(); got != tc.wantCode {
				t.Errorf("Code() = %v, want %v", got, tc.wantCode)
			}
		})
	}

	t.Run("a domain error carries a detail", func(t *testing.T) {
		result := connectErrorFrom(&platform.DomainError{Code: platform.ErrorCode([[.Name]]def.ErrorCodeInvalidLabel), Message: "x"})
		if len(result.Details()) == 0 {
			t.Error("expected at least one error detail")
		}
	})
}
