[[if ne .Shape "gateway" -]]
package connect

import (
	"testing"
	"time"

	"connectrpc.com/connect"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.ModulePath]]/internal/domain/[[.Entity]]"
	"[[.ModulePath]]/internal/testhelpers"
)

[[- if eq .Shape "read-model"]]

func newTest[[.Entity | pascal]]Handler(t *testing.T) (*[[.Entity | pascal]]Handler, *testhelpers.InMemory[[.Entity | pascal]]Repo) {
	t.Helper()
	svc, repo := testhelpers.NewTest[[.Entity | pascal]]Service(t)

	return New[[.Entity | pascal]]Handler(svc), repo
}
[[- else]]

func newTest[[.Entity | pascal]]Handler(t *testing.T) (*[[.Entity | pascal]]Handler, *testhelpers.InMemory[[.Entity | pascal]]Repo, *testhelpers.RecordingEventDispatcher) {
	t.Helper()
	svc, repo, dispatcher := testhelpers.NewTest[[.Entity | pascal]]Service(t)

	return New[[.Entity | pascal]]Handler(svc), repo, dispatcher
}
[[- end]]

func Test[[.Entity | pascal]]Handler_Get[[.Entity | pascal]](t *testing.T) {
[[- if eq .Shape "read-model"]]
	handler, repo := newTest[[.Entity | pascal]]Handler(t)
	created := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	repo.Seed(&[[.Entity]].[[.Entity | pascal]]{
		ID: 42, Label: [[.Entity]].ReconstituteLabel("Première"), CreatedAt: created, UpdatedAt: created,
	})
[[- else]]
	handler, repo, _ := newTest[[.Entity | pascal]]Handler(t)
	created := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	repo.Seed([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{ID: 42, Label: "Première", CreatedAt: created, UpdatedAt: created}))
[[- end]]

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

[[- if eq .Shape "aggregate"]]

func Test[[.Entity | pascal]]Handler_Delete[[.Entity | pascal]](t *testing.T) {
	handler, repo, dispatcher := newTest[[.Entity | pascal]]Handler(t)
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
[[- end]]
[[- else -]]
package connect

import (
	"testing"

	"connectrpc.com/connect"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.ModulePath]]/internal/domain/[[.Entity]]"
	"[[.ModulePath]]/internal/testhelpers"
)

func newTest[[.Entity | pascal]]Handler(t *testing.T) (*[[.Entity | pascal]]Handler, *testhelpers.Fake[[.Entity | pascal]]Gateway, *testhelpers.RecordingEventPublisher) {
	t.Helper()
	svc, gateway, publisher := testhelpers.NewTest[[.Entity | pascal]]Service(t)

	return New[[.Entity | pascal]]Handler(svc), gateway, publisher
}

func Test[[.Entity | pascal]]Handler_Send[[.Entity | pascal]](t *testing.T) {
	t.Run("sends and publishes the Sent event", func(t *testing.T) {
		handler, gateway, publisher := newTest[[.Entity | pascal]]Handler(t)
		_, err := handler.Send[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: "Bonjour"}))
		if err != nil {
			t.Fatalf("Send[[.Entity | pascal]]() unexpected error: %v", err)
		}
		if len(gateway.Sent) != 1 || len(publisher.Events) != 1 {
			t.Errorf("sent/events = %d/%d, want 1/1", len(gateway.Sent), len(publisher.Events))
		}
	})

	t.Run("empty label is CodeInvalidArgument with the InvalidLabel detail", func(t *testing.T) {
		handler, _, _ := newTest[[.Entity | pascal]]Handler(t)
		_, err := handler.Send[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: ""}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("CodeOf(err) = %v, want CodeInvalidArgument", connect.CodeOf(err))
		}
		if de := domainErrorDetail(t, err); de.GetCode() != int32([[.Name]]def.ErrorCodeInvalidLabel) {
			t.Errorf("detail code = %d", de.GetCode())
		}
	})

	t.Run("a failing gateway is CodeUnavailable", func(t *testing.T) {
		handler, gateway, _ := newTest[[.Entity | pascal]]Handler(t)
		gateway.Err = [[.Entity]].ErrGatewayUnavailable
		_, err := handler.Send[[.Entity | pascal]](authedCtx(), connect.NewRequest(&[[.Name]]v1.Send[[.Entity | pascal]]Request{Label: "Bonjour"}))
		if connect.CodeOf(err) != connect.CodeUnavailable {
			t.Errorf("CodeOf(err) = %v, want CodeUnavailable", connect.CodeOf(err))
		}
	})
}
[[- end]]
