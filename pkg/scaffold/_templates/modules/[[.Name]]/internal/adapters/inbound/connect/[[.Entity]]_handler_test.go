[[if .Database -]]
package connect

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"[[.PlatformPath]]/pkg/platform/authctx"
	commonv1 "[[.ContractsPath]]/go/network/common/v1"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
	"[[.OrgPrefix]]/[[.Name]]/internal/testhelpers"
)

func newTestHandler(t *testing.T) (*[[.Entity | pascal]]Handler, *testhelpers.InMemory[[.Entity | pascal]]Repo, *testhelpers.RecordingEventDispatcher) {
	t.Helper()
	svc, repo, dispatcher := testhelpers.NewTest[[.Entity | pascal]]Service(t)

	return New[[.Entity | pascal]]Handler(svc), repo, dispatcher
}

func authedCtx() context.Context {
	return authctx.WithUserID(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000001"))
}

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

func Test[[.Entity | pascal]]Handler_Get[[.Entity | pascal]](t *testing.T) {
	handler, repo, _ := newTestHandler(t)
	created := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	repo.Seed([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{ID: 42, Label: "Première", CreatedAt: created, UpdatedAt: created}))

	t.Run("returns the mapped [[.Entity]]", func(t *testing.T) {
		resp, err := handler.Get[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Get[[.Entity | pascal]]Request{Id: 42}))
		if err != nil {
			t.Fatalf("Get[[.Entity | pascal]]() unexpected error: %v", err)
		}
		got := resp.Msg.Get[[.Entity | pascal]]()
		if got.GetId() != 42 || got.GetLabel() != "Première" || !got.GetCreatedAt().AsTime().Equal(created) {
			t.Errorf("got %d/%q/%v", got.GetId(), got.GetLabel(), got.GetCreatedAt().AsTime())
		}
	})

	t.Run("unknown id is CodeNotFound with the NotFound detail", func(t *testing.T) {
		_, err := handler.Get[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Get[[.Entity | pascal]]Request{Id: 999}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("CodeOf(err) = %v, want CodeNotFound", connect.CodeOf(err))
		}
		if de := domainErrorDetail(t, err); de.GetCode() != int32([[.Name]]def.ErrorCodeNotFound) {
			t.Errorf("detail code = %d, want %d", de.GetCode(), int32([[.Name]]def.ErrorCodeNotFound))
		}
	})

	t.Run("id 0 is CodeInvalidArgument with the InvalidID detail", func(t *testing.T) {
		_, err := handler.Get[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Get[[.Entity | pascal]]Request{Id: 0}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("CodeOf(err) = %v, want CodeInvalidArgument", connect.CodeOf(err))
		}
		if de := domainErrorDetail(t, err); de.GetCode() != int32([[.Name]]def.ErrorCodeInvalidID) {
			t.Errorf("detail code = %d, want %d", de.GetCode(), int32([[.Name]]def.ErrorCodeInvalidID))
		}
	})
}

func Test[[.Entity | pascal]]Handler_Delete[[.Entity | pascal]](t *testing.T) {
	handler, repo, dispatcher := newTestHandler(t)
	repo.Seed([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{ID: 42, Label: "Première"}))

	t.Run("deletes the row and publishes the Deleted event", func(t *testing.T) {
		_, err := handler.Delete[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Delete[[.Entity | pascal]]Request{Id: 42}))
		if err != nil {
			t.Fatalf("Delete[[.Entity | pascal]]() unexpected error: %v", err)
		}
		if repo.Len() != 0 || len(dispatcher.Events) != 1 {
			t.Errorf("rows/events = %d/%d, want 0/1", repo.Len(), len(dispatcher.Events))
		}
	})

	t.Run("deleting it again is CodeNotFound", func(t *testing.T) {
		_, err := handler.Delete[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Delete[[.Entity | pascal]]Request{Id: 42}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("CodeOf(err) = %v, want CodeNotFound", connect.CodeOf(err))
		}
	})
}
[[- else -]]
package connect

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"[[.PlatformPath]]/pkg/platform/authctx"
	commonv1 "[[.ContractsPath]]/go/network/common/v1"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
	"[[.OrgPrefix]]/[[.Name]]/internal/testhelpers"
)

func newTestHandler(t *testing.T) (*[[.Entity | pascal]]Handler, *testhelpers.Fake[[.Entity | pascal]]Gateway, *testhelpers.RecordingEventPublisher) {
	t.Helper()
	svc, gateway, publisher := testhelpers.NewTest[[.Entity | pascal]]Service(t)

	return New[[.Entity | pascal]]Handler(svc), gateway, publisher
}

func authedCtx() context.Context {
	return authctx.WithUserID(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000001"))
}

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

func Test[[.Entity | pascal]]Handler_Send[[.Entity | pascal]](t *testing.T) {
	t.Run("sends and publishes the Sent event", func(t *testing.T) {
		handler, gateway, publisher := newTestHandler(t)
		_, err := handler.Send[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: "Bonjour"}))
		if err != nil {
			t.Fatalf("Send[[.Entity | pascal]]() unexpected error: %v", err)
		}
		if len(gateway.Sent) != 1 || len(publisher.Events) != 1 {
			t.Errorf("sent/events = %d/%d, want 1/1", len(gateway.Sent), len(publisher.Events))
		}
	})

	t.Run("empty label is CodeInvalidArgument with the InvalidLabel detail", func(t *testing.T) {
		handler, _, _ := newTestHandler(t)
		_, err := handler.Send[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: ""}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("CodeOf(err) = %v, want CodeInvalidArgument", connect.CodeOf(err))
		}
		if de := domainErrorDetail(t, err); de.GetCode() != int32([[.Name]]def.ErrorCodeInvalidLabel) {
			t.Errorf("detail code = %d", de.GetCode())
		}
	})

	t.Run("a failing gateway is CodeUnavailable", func(t *testing.T) {
		handler, gateway, _ := newTestHandler(t)
		gateway.Err = [[.Entity]].ErrGatewayUnavailable
		_, err := handler.Send[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: "Bonjour"}))
		if connect.CodeOf(err) != connect.CodeUnavailable {
			t.Errorf("CodeOf(err) = %v, want CodeUnavailable", connect.CodeOf(err))
		}
	})
}
[[- end]]
