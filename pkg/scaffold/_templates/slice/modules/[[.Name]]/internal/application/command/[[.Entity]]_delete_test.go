package command_test

import (
	"context"
	"errors"
	"testing"

	"[[.ModulePath]]/internal/application/command"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
	"[[.ModulePath]]/internal/testhelpers"
)

func TestDelete[[.Entity | pascal]]Command_Execute(t *testing.T) {
	t.Run("deletes the row and dispatches one Deleted event", func(t *testing.T) {
		repo := testhelpers.NewInMemory[[.Entity | pascal]]Repo()
		repo.Seed([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{ID: 42, Label: "Première"}))
		dispatcher := &testhelpers.RecordingEventDispatcher{}
		cmd := command.NewDelete[[.Entity | pascal]]Command(repo, testhelpers.PassthroughUoW{}, dispatcher)

		if err := cmd.Execute(testhelpers.AuthedCtx(), 42); err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
		}
		if repo.Len() != 0 {
			t.Errorf("repo still holds %d rows", repo.Len())
		}
		if len(dispatcher.Events) != 1 || dispatcher.Events[0].EventType() != [[.Entity]].EventTypeDeleted {
			t.Fatalf("dispatched = %v, want one Deleted event", dispatcher.Events)
		}
		if dispatcher.Events[0].GetUserID() != testhelpers.TestUser || dispatcher.Events[0].GetAggregateID() != "42" {
			t.Errorf("event user/aggregate = %s/%s", dispatcher.Events[0].GetUserID(), dispatcher.Events[0].GetAggregateID())
		}
	})

	t.Run("returns [[.Entity]].ErrNotFound and dispatches nothing for an unknown id", func(t *testing.T) {
		repo := testhelpers.NewInMemory[[.Entity | pascal]]Repo()
		dispatcher := &testhelpers.RecordingEventDispatcher{}
		cmd := command.NewDelete[[.Entity | pascal]]Command(repo, testhelpers.PassthroughUoW{}, dispatcher)

		err := cmd.Execute(testhelpers.AuthedCtx(), 999)
		if !errors.Is(err, [[.Entity]].ErrNotFound) {
			t.Errorf("Execute() error = %v, want ErrNotFound", err)
		}
		if len(dispatcher.Events) != 0 {
			t.Errorf("dispatched %d events, want 0", len(dispatcher.Events))
		}
	})

	t.Run("returns [[.Entity]].ErrInvalidID before touching the repository", func(t *testing.T) {
		repo := testhelpers.NewInMemory[[.Entity | pascal]]Repo()
		cmd := command.NewDelete[[.Entity | pascal]]Command(repo, testhelpers.PassthroughUoW{}, testhelpers.NoopEventDispatcher{})

		if err := cmd.Execute(testhelpers.AuthedCtx(), 0); !errors.Is(err, [[.Entity]].ErrInvalidID) {
			t.Errorf("Execute() error = %v, want ErrInvalidID", err)
		}
	})

	t.Run("fails without an authenticated user", func(t *testing.T) {
		repo := testhelpers.NewInMemory[[.Entity | pascal]]Repo()
		repo.Seed([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{ID: 42, Label: "Première"}))
		cmd := command.NewDelete[[.Entity | pascal]]Command(repo, testhelpers.PassthroughUoW{}, testhelpers.NoopEventDispatcher{})

		if err := cmd.Execute(context.Background(), 42); err == nil {
			t.Error("Execute() without user expected an error, got nil")
		}
		if repo.Len() != 1 {
			t.Errorf("repo holds %d rows, want 1 (nothing deleted)", repo.Len())
		}
	})
}
