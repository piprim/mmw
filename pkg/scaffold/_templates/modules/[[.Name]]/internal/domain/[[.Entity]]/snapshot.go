package [[.Entity]]

import (
	"time"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// Snapshot is the Memento of the [[.Entity | pascal]] aggregate: its persistent
// state as primitives, so adapters persist and reconstitute it without
// reaching into private fields. It carries no storage metadata and no events.
type Snapshot struct {
	ID        int64
	Label     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Snapshot returns the current state of the aggregate.
func (a *[[.Entity | pascal]]) Snapshot() Snapshot {
	return Snapshot{
		ID:        a.id.Int64(),
		Label:     a.label.String(),
		CreatedAt: a.createdAt,
		UpdatedAt: a.updatedAt,
	}
}

// Reconstitute rebuilds a [[.Entity | pascal]] from stored data. Value objects are
// reconstituted, not validated: the database is the source of truth.
func Reconstitute(snap *Snapshot) *[[.Entity | pascal]] {
	return &[[.Entity | pascal]]{
		id:        ID(snap.ID),
		label:     ReconstituteLabel(snap.Label),
		createdAt: snap.CreatedAt,
		updatedAt: snap.UpdatedAt,
		events:    []domain.Event{},
	}
}
