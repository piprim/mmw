package connect

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	commonv1 "[[.ContractsPath]]/go/network/common/v1"

	"[[.ModulePath]]/internal/testhelpers"
)

// Helpers shared by every slice's handler test.

func authedCtx() context.Context { return testhelpers.AuthedCtx() }

// domainErrorDetail extracts the commonv1.DomainError detail attached to a connect error.
func domainErrorDetail(t *testing.T, err error) *commonv1.DomainError {
	t.Helper()
	connectErr, ok := errors.AsType[*connect.Error](err)
	if !ok {
		t.Fatalf("expected *connect.Error, got %T", err)
	}
	for _, d := range connectErr.Details() {
		msg, err := d.Value()
		if err != nil {
			continue
		}
		if de, ok := msg.(*commonv1.DomainError); ok {
			return de
		}
	}
	t.Fatal("no commonv1.DomainError detail attached")

	return nil
}
