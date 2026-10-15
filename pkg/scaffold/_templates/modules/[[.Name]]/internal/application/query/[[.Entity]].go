package query

import (
	"context"

	"github.com/rotisserie/eris"

	"[[.OrgPrefix]]/[[.Name]]/internal/application/ports"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// Get[[.Entity | pascal]]Query retrieves a single [[.Entity]] by ID.
//
// It does not read the user from the context: a [[.Entity]] has no owner, so
// authentication is left to the transport (bearer middleware).
type Get[[.Entity | pascal]]Query struct {
	repository ports.[[.Entity | pascal]]Repository
}

// NewGet[[.Entity | pascal]]Query creates a new Get[[.Entity | pascal]]Query.
func NewGet[[.Entity | pascal]]Query(repository ports.[[.Entity | pascal]]Repository) *Get[[.Entity | pascal]]Query {
	return &Get[[.Entity | pascal]]Query{repository: repository}
}

// Execute retrieves a [[.Entity]] by its integer primary key.
func (q *Get[[.Entity | pascal]]Query) Execute(ctx context.Context, id int64) (*[[.Entity]].[[.Entity | pascal]], error) {
	[[.Entity]]ID, err := [[.Entity]].ParseID(id)
	if err != nil {
		return nil, eris.Wrap(err, "invalid [[.Entity]] ID")
	}

	a, err := q.repository.[[.Entity | pascal]]FindByID(ctx, [[.Entity]]ID)
	if err != nil {
		return nil, eris.Wrap(err, "finding [[.Entity]]")
	}

	return a, nil
}
