package ports

import (
	"context"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// EventPublisher publishes domain events straight to the event bus. There is
// no outbox without a database: delivery is at-most-once.
type EventPublisher interface {
	// Publish sends the events to the bus, in order; it stops at the first failure.
	Publish(ctx context.Context, events []domain.Event) error
}
