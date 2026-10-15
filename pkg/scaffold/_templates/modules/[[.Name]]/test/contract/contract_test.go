[[if .Database -]]
// Package contract tests that inproc.[[.Entity | pascal]]ContractAdapter implements the
// [[.Name]]def.[[.Entity | pascal]]Service contract: mapping and error propagation,
// with no infrastructure.
package contract

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"[[.PlatformPath]]/pkg/platform"
	"[[.PlatformPath]]/pkg/platform/authctx"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/inproc"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
	"[[.OrgPrefix]]/[[.Name]]/internal/testhelpers"
)

func newAdapter(t *testing.T) ([[.Name]]def.[[.Entity | pascal]]Service, *testhelpers.InMemory[[.Entity | pascal]]Repo) {
	t.Helper()
	svc, repo, _ := testhelpers.NewTest[[.Entity | pascal]]Service(t)

	return inproc.New[[.Entity | pascal]]ContractAdapter(svc), repo
}

func authedCtx() context.Context {
	return authctx.WithUserID(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000001"))
}

// hasDomainCode reports whether err carries the given contract error code.
// The adapter wraps a *platform.DomainError, so match by code.
func hasDomainCode(err error, want platform.ErrorCode) bool {
	de, ok := errors.AsType[*platform.DomainError](err)

	return ok && de.Code == want
}

func TestContract_[[.Entity | pascal]]Service(t *testing.T) {
	adapter, repo := newAdapter(t)
	repo.Seed([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{ID: 42, Label: "Première"}))
	ctx := authedCtx()

	t.Run("Get returns the mapped [[.Entity]]", func(t *testing.T) {
		resp, err := adapter.Get[[.Entity | pascal]](ctx, &[[.Name]]v1.Get[[.Entity | pascal]]Request{Id: 42})
		if err != nil {
			t.Fatalf("Get[[.Entity | pascal]]() error: %v", err)
		}
		if resp.Get[[.Entity | pascal]]().GetId() != 42 || resp.Get[[.Entity | pascal]]().GetLabel() != "Première" {
			t.Errorf("got %v", resp.Get[[.Entity | pascal]]())
		}
	})

	t.Run("Get of an unknown id carries ErrorCodeNotFound", func(t *testing.T) {
		_, err := adapter.Get[[.Entity | pascal]](ctx, &[[.Name]]v1.Get[[.Entity | pascal]]Request{Id: 999})
		if !hasDomainCode(err, platform.ErrorCode([[.Name]]def.ErrorCodeNotFound)) {
			t.Errorf("error = %v, want ErrorCodeNotFound", err)
		}
	})

	t.Run("Get of id 0 carries ErrorCodeInvalidID", func(t *testing.T) {
		_, err := adapter.Get[[.Entity | pascal]](ctx, &[[.Name]]v1.Get[[.Entity | pascal]]Request{Id: 0})
		if !hasDomainCode(err, platform.ErrorCode([[.Name]]def.ErrorCodeInvalidID)) {
			t.Errorf("error = %v, want ErrorCodeInvalidID", err)
		}
	})

	t.Run("Delete removes the row, a second Delete carries ErrorCodeNotFound", func(t *testing.T) {
		if _, err := adapter.Delete[[.Entity | pascal]](ctx, &[[.Name]]v1.Delete[[.Entity | pascal]]Request{Id: 42}); err != nil {
			t.Fatalf("Delete[[.Entity | pascal]]() error: %v", err)
		}
		_, err := adapter.Delete[[.Entity | pascal]](ctx, &[[.Name]]v1.Delete[[.Entity | pascal]]Request{Id: 42})
		if !hasDomainCode(err, platform.ErrorCode([[.Name]]def.ErrorCodeNotFound)) {
			t.Errorf("error = %v, want ErrorCodeNotFound", err)
		}
	})
}
[[- else -]]
// Package contract tests that inproc.[[.Entity | pascal]]ContractAdapter implements the
// [[.Name]]def.[[.Entity | pascal]]Service contract: mapping and error propagation,
// with no infrastructure.
package contract

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"[[.PlatformPath]]/pkg/platform"
	"[[.PlatformPath]]/pkg/platform/authctx"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/inproc"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
	"[[.OrgPrefix]]/[[.Name]]/internal/testhelpers"
)

func newAdapter(t *testing.T) ([[.Name]]def.[[.Entity | pascal]]Service, *testhelpers.Fake[[.Entity | pascal]]Gateway) {
	t.Helper()
	svc, gateway, _ := testhelpers.NewTest[[.Entity | pascal]]Service(t)

	return inproc.New[[.Entity | pascal]]ContractAdapter(svc), gateway
}

func authedCtx() context.Context {
	return authctx.WithUserID(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000001"))
}

// hasDomainCode reports whether err carries the given contract error code.
// The adapter wraps a *platform.DomainError, so match by code.
func hasDomainCode(err error, want platform.ErrorCode) bool {
	de, ok := errors.AsType[*platform.DomainError](err)

	return ok && de.Code == want
}

func TestContract_[[.Entity | pascal]]Service(t *testing.T) {
	ctx := authedCtx()

	t.Run("Send hands the label to the gateway", func(t *testing.T) {
		svc, gateway := newAdapter(t)
		if _, err := svc.Send[[.Entity | pascal]](ctx, &[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: "Bonjour"}); err != nil {
			t.Fatalf("Send[[.Entity | pascal]]() error: %v", err)
		}
		if len(gateway.Sent) != 1 || gateway.Sent[0].Label().String() != "Bonjour" {
			t.Errorf("gateway.Sent = %v", gateway.Sent)
		}
	})

	t.Run("an empty label carries ErrorCodeInvalidLabel", func(t *testing.T) {
		svc, _ := newAdapter(t)
		_, err := svc.Send[[.Entity | pascal]](ctx, &[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: ""})
		if !hasDomainCode(err, platform.ErrorCode([[.Name]]def.ErrorCodeInvalidLabel)) {
			t.Errorf("error = %v, want ErrorCodeInvalidLabel", err)
		}
	})

	t.Run("a failing gateway carries ErrorCodeGatewayUnavailable", func(t *testing.T) {
		svc, gateway := newAdapter(t)
		gateway.Err = [[.Entity]].ErrGatewayUnavailable
		_, err := svc.Send[[.Entity | pascal]](ctx, &[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: "Bonjour"})
		if !hasDomainCode(err, platform.ErrorCode([[.Name]]def.ErrorCodeGatewayUnavailable)) {
			t.Errorf("error = %v, want ErrorCodeGatewayUnavailable", err)
		}
	})
}
[[- end]]
