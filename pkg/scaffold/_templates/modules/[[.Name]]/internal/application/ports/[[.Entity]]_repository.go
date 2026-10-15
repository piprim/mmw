package ports

import (
	"context"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// [[.Entity | pascal]]Repository is the secondary port of the [[.Entity]] slice:
// needed by the application, implemented by adapters. Methods are prefixed
// with the entity so a [[.Entity | pascal]]Save can join them without renaming.
type [[.Entity | pascal]]Repository interface {
	// [[.Entity | pascal]]FindByID retrieves a [[.Entity]] by its integer primary key.
	// Returns [[.Entity]].ErrNotFound if no row matches.
	[[.Entity | pascal]]FindByID(ctx context.Context, id [[.Entity]].ID) (*[[.Entity]].[[.Entity | pascal]], error)

	// [[.Entity | pascal]]Delete removes a [[.Entity]].
	// Returns [[.Entity]].ErrNotFound if no row matches.
	[[.Entity | pascal]]Delete(ctx context.Context, id [[.Entity]].ID) error

	// Health reports whether the storage answers (row count).
	Health(ctx context.Context) (any, error)
}
