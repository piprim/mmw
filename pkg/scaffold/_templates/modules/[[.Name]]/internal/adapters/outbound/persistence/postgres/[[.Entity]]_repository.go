package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/piprim/boa"
	"github.com/rotisserie/eris"

	pfuow "[[.PlatformPath]]/pkg/platform/pg/uow"
	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/outbound/persistence/postgres/dao/boamodel"
	"[[.OrgPrefix]]/[[.Name]]/internal/application/ports"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// Postgres[[.Entity | pascal]]Repository implements ports.[[.Entity | pascal]]Repository on
// [[.Name]].[[.Entity]] through boa. boa runs on the executor of the unit of
// work, so a query sees the transaction carried by ctx. A sqlc query does
// the same through sqlc.New(r.uow.Executor(ctx)), built per call.
type Postgres[[.Entity | pascal]]Repository struct {
	boa *boa.DB
}

var _ ports.[[.Entity | pascal]]Repository = (*Postgres[[.Entity | pascal]]Repository)(nil)

// NewPostgres[[.Entity | pascal]]Repository creates the repository. The boa DB has no
// pool of its own: every query goes through the unit of work.
func NewPostgres[[.Entity | pascal]]Repository(uow *pfuow.UnitOfWork) *Postgres[[.Entity | pascal]]Repository {
	return &Postgres[[.Entity | pascal]]Repository{
		boa: boa.New(nil, boa.WithExecutorResolver(func(ctx context.Context) boa.DBExecutor {
			return uow.Executor(ctx)
		})),
	}
}

// [[.Entity | pascal]]FindByID retrieves a [[.Entity]] by its integer primary key.
func (r *Postgres[[.Entity | pascal]]Repository) [[.Entity | pascal]]FindByID(
	ctx context.Context, id [[.Entity]].ID,
) (*[[.Entity]].[[.Entity | pascal]], error) {
	m := boamodel.[[.Name | pascal]][[.Entity | pascal]]{ID: id.Int64()}

	err := r.boa.NewSelect().
		Model(&m).
		Column([[.Entity]]Columns...).
		WherePK().
		Scan(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, [[.Entity]].ErrNotFound
		}

		return nil, eris.Wrap(err, "querying [[.Entity]]")
	}

	snap := [[.Entity]]FromModel(&m)

	return [[.Entity]].Reconstitute(&snap), nil
}

// [[.Entity | pascal]]Delete removes a [[.Entity]].
func (r *Postgres[[.Entity | pascal]]Repository) [[.Entity | pascal]]Delete(ctx context.Context, id [[.Entity]].ID) error {
	m := boamodel.[[.Name | pascal]][[.Entity | pascal]]{ID: id.Int64()}

	result, err := r.boa.NewDelete().Model(&m).WherePK().Exec(ctx)
	if err != nil {
		return eris.Wrap(err, "deleting [[.Entity]]")
	}

	if result.RowsAffected() == 0 {
		return [[.Entity]].ErrNotFound
	}

	return nil
}

// Health counts the rows of [[.Name]].[[.Entity]].
func (r *Postgres[[.Entity | pascal]]Repository) Health(ctx context.Context) (any, error) {
	count, err := r.boa.NewSelect().Model((*boamodel.[[.Name | pascal]][[.Entity | pascal]])(nil)).Count(ctx)
	if err != nil {
		return 0, eris.Wrap(err, "counting [[.Entity]]s")
	}

	return int(count), nil
}
