package postgres

import (
	"testing"
	"time"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/outbound/persistence/postgres/dao/boamodel"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

func Test[[.Entity | pascal]]RowMapping(t *testing.T) {
	created := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	m := boamodel.[[.Name | pascal]][[.Entity | pascal]]{ID: 42, Label: "Première", CreatedAt: created, UpdatedAt: created.Add(time.Hour)}
	want := [[.Entity]].Snapshot{ID: 42, Label: "Première", CreatedAt: created, UpdatedAt: created.Add(time.Hour)}

	t.Run("model to snapshot maps every field", func(t *testing.T) {
		if got := [[.Entity]]FromModel(&m); got != want {
			t.Errorf("[[.Entity]]FromModel() = %+v, want %+v", got, want)
		}
	})

	t.Run("snapshot to model round-trips", func(t *testing.T) {
		if got := [[.Entity]]ToModel(&want); got.ID != m.ID || got.Label != m.Label || !got.CreatedAt.Equal(m.CreatedAt) || !got.UpdatedAt.Equal(m.UpdatedAt) {
			t.Errorf("[[.Entity]]ToModel() = %+v, want %+v", got, m)
		}
	})

	t.Run("columns name the four columns of the table", func(t *testing.T) {
		if len([[.Entity]]Columns) != 4 || [[.Entity]]Columns[1] != "label" {
			t.Errorf("[[.Entity]]Columns = %v", [[.Entity]]Columns)
		}
	})
}
