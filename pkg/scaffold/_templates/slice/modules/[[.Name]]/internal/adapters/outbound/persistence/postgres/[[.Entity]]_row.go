package postgres

import (
	"[[.ModulePath]]/internal/adapters/outbound/persistence/postgres/dao/boamodel"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

// [[.Entity]]Columns are the columns of [[.Name]].[[.Entity]] read into the domain.
// Column names come only from the generated code.
var [[.Entity]]Columns = func() []string {
	c := boamodel.Columns.[[.Name | pascal]][[.Entity | pascal]]

	return []string{c.ID, c.Label, c.CreatedAt, c.UpdatedAt}
}()

[[- if eq .Shape "read-model"]]

// [[.Entity]]FromModel maps the boa model to the read model. ReconstituteLabel
// (not NewLabel) trusts the stored value: the database is the source of truth.
func [[.Entity]]FromModel(m *boamodel.[[.Name | pascal]][[.Entity | pascal]]) *[[.Entity]].[[.Entity | pascal]] {
	return &[[.Entity]].[[.Entity | pascal]]{
		ID:        [[.Entity]].ID(m.ID),
		Label:     [[.Entity]].ReconstituteLabel(m.Label),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
[[- else]]

// [[.Entity]]FromModel maps the boa model to the aggregate's snapshot.
// Reconstitute (not NewLabel) then trusts the stored value.
func [[.Entity]]FromModel(m *boamodel.[[.Name | pascal]][[.Entity | pascal]]) [[.Entity]].Snapshot {
	return [[.Entity]].Snapshot{
		ID:        m.ID,
		Label:     m.Label,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// [[.Entity]]ToModel maps a snapshot to the boa model, for the insert and
// update the create command will need.
//
//nolint:unused // ready for the create command; the row test covers it
func [[.Entity]]ToModel(snap *[[.Entity]].Snapshot) boamodel.[[.Name | pascal]][[.Entity | pascal]] {
	return boamodel.[[.Name | pascal]][[.Entity | pascal]]{
		ID:        snap.ID,
		Label:     snap.Label,
		CreatedAt: snap.CreatedAt,
		UpdatedAt: snap.UpdatedAt,
	}
}
[[- end]]
