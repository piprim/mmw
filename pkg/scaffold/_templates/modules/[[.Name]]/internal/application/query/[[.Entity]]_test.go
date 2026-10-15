package query_test

import (
	"errors"
	"testing"

	"[[.OrgPrefix]]/[[.Name]]/internal/application/query"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
	"[[.OrgPrefix]]/[[.Name]]/internal/testhelpers"
)

func TestGet[[.Entity | pascal]]Query_Execute(t *testing.T) {
	repo := testhelpers.NewInMemory[[.Entity | pascal]]Repo()
	repo.Seed([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{ID: 42, Label: "Première"}))
	q := query.NewGet[[.Entity | pascal]]Query(repo)

	t.Run("returns the [[.Entity]] for an existing id", func(t *testing.T) {
		got, err := q.Execute(t.Context(), 42)
		if err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
		}
		if got.ID() != 42 || got.Label().String() != "Première" {
			t.Errorf("got %d/%q, want 42/Première", got.ID(), got.Label())
		}
	})

	t.Run("returns [[.Entity]].ErrNotFound for an unknown id", func(t *testing.T) {
		_, err := q.Execute(t.Context(), 999)
		if !errors.Is(err, [[.Entity]].ErrNotFound) {
			t.Errorf("Execute() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("returns [[.Entity]].ErrInvalidID for a non-positive id", func(t *testing.T) {
		_, err := q.Execute(t.Context(), 0)
		if !errors.Is(err, [[.Entity]].ErrInvalidID) {
			t.Errorf("Execute() error = %v, want ErrInvalidID", err)
		}
	})
}
