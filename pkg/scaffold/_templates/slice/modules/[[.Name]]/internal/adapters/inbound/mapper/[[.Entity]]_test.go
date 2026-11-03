package mapper_test

import (
	"testing"
	"time"

	"[[.ModulePath]]/internal/adapters/inbound/mapper"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

func Test[[.Entity | pascal]]ToProto(t *testing.T) {
	created := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
[[- if eq .Shape "read-model"]]
	p := mapper.[[.Entity | pascal]]ToProto(&[[.Entity]].[[.Entity | pascal]]{
		ID: 42, Label: [[.Entity]].ReconstituteLabel("Première"), CreatedAt: created, UpdatedAt: updated,
	})
[[- else]]
	p := mapper.[[.Entity | pascal]]ToProto([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{
		ID: 42, Label: "Première", CreatedAt: created, UpdatedAt: updated,
	}))
[[- end]]

	t.Run("maps id and label", func(t *testing.T) {
		if p.GetId() != 42 || p.GetLabel() != "Première" {
			t.Errorf("Id/Label = %d/%q", p.GetId(), p.GetLabel())
		}
	})

	t.Run("maps both timestamps", func(t *testing.T) {
		if !p.GetCreatedAt().AsTime().Equal(created) || !p.GetUpdatedAt().AsTime().Equal(updated) {
			t.Errorf("CreatedAt/UpdatedAt = %v/%v", p.GetCreatedAt().AsTime(), p.GetUpdatedAt().AsTime())
		}
	})
}
