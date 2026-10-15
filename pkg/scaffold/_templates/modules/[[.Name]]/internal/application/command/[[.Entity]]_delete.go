package command

import (
	"context"

	"github.com/rotisserie/eris"

	"[[.PlatformPath]]/pkg/platform/authctx"
	"[[.OrgPrefix]]/[[.Name]]/internal/application/ports"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// Delete[[.Entity | pascal]]Command deletes a [[.Entity]] and emits its Deleted event.
// The row deletion and the outbox row land in one transaction: the event is
// published if and only if the row is gone.
type Delete[[.Entity | pascal]]Command struct {
	repository      ports.[[.Entity | pascal]]Repository
	unitOfWork      ports.UnitOfWork
	eventDispatcher ports.EventDispatcher
}

// NewDelete[[.Entity | pascal]]Command creates a new Delete[[.Entity | pascal]]Command.
func NewDelete[[.Entity | pascal]]Command(
	repository ports.[[.Entity | pascal]]Repository,
	unitOfWork ports.UnitOfWork,
	eventDispatcher ports.EventDispatcher,
) *Delete[[.Entity | pascal]]Command {
	return &Delete[[.Entity | pascal]]Command{
		repository:      repository,
		unitOfWork:      unitOfWork,
		eventDispatcher: eventDispatcher,
	}
}

// Execute deletes the [[.Entity]] with the given ID.
// Input is validated before the transaction opens, so a bad request never
// holds a connection.
func (c *Delete[[.Entity | pascal]]Command) Execute(ctx context.Context, id int64) error {
	userID, err := authctx.UserIDFromContext(ctx)
	if err != nil {
		return eris.Wrap(err, "delete [[.Entity]]")
	}

	[[.Entity]]ID, err := [[.Entity]].ParseID(id)
	if err != nil {
		return eris.Wrap(err, "invalid [[.Entity]] ID")
	}

	err = c.unitOfWork.WithTransaction(ctx, func(txCtx context.Context) error {
		a, err := c.repository.[[.Entity | pascal]]FindByID(txCtx, [[.Entity]]ID)
		if err != nil {
			return eris.Wrap(err, "finding [[.Entity]] to delete")
		}

		a.Delete(userID)

		if err := c.repository.[[.Entity | pascal]]Delete(txCtx, [[.Entity]]ID); err != nil {
			return eris.Wrap(err, "deleting [[.Entity]]")
		}

		if err := c.eventDispatcher.Dispatch(txCtx, a.Events()); err != nil {
			return eris.Wrap(err, "dispatching events")
		}

		a.ClearEvents()

		return nil
	})

	return eris.Wrap(err, "uow execution failed")
}
