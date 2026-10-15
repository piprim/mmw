package command

import (
	"context"

	"github.com/rotisserie/eris"

	"[[.PlatformPath]]/pkg/platform/authctx"
	"[[.OrgPrefix]]/[[.Name]]/internal/application/ports"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// Send[[.Entity | pascal]]Command validates a label, hands the [[.Entity]] to the
// gateway and publishes its Sent event. The event is published only after
// the gateway accepted the message.
type Send[[.Entity | pascal]]Command struct {
	gateway   ports.[[.Entity | pascal]]Gateway
	publisher ports.EventPublisher
}

// NewSend[[.Entity | pascal]]Command creates a new Send[[.Entity | pascal]]Command.
func NewSend[[.Entity | pascal]]Command(
	gateway ports.[[.Entity | pascal]]Gateway, publisher ports.EventPublisher,
) *Send[[.Entity | pascal]]Command {
	return &Send[[.Entity | pascal]]Command{gateway: gateway, publisher: publisher}
}

// Execute sends a [[.Entity]] with the given label.
func (c *Send[[.Entity | pascal]]Command) Execute(ctx context.Context, label string) error {
	userID, err := authctx.UserIDFromContext(ctx)
	if err != nil {
		return eris.Wrap(err, "send [[.Entity]]")
	}

	lbl, err := [[.Entity]].NewLabel(label)
	if err != nil {
		return eris.Wrap(err, "invalid label")
	}

	a := [[.Entity]].New(lbl)
	if err := c.gateway.[[.Entity | pascal]]Send(ctx, a); err != nil {
		return eris.Wrap(err, "sending [[.Entity]]")
	}

	a.MarkSent(userID)
	// ponytail: the provider already accepted the message; a caller that retries on
	// this error sends it twice. At-most-once is the documented contract,
	// log-and-continue is the upgrade.
	if err := c.publisher.Publish(ctx, a.Events()); err != nil {
		return eris.Wrap(err, "publishing events")
	}

	a.ClearEvents()

	return nil
}
