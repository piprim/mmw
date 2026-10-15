package mapper_test

import (
	"testing"
	"time"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/mapper"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

func Test[[.Entity | pascal]]ToProto(t *testing.T) {
	created := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	p := mapper.[[.Entity | pascal]]ToProto([[.Entity]].Reconstitute(&[[.Entity]].Snapshot{
		ID: 42, Label: "Première", CreatedAt: created, UpdatedAt: updated,
	}))

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
