package ports

import (
	"context"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// EventDispatcher defines the interface for publishing domain events.
// This is a secondary port (driven): needed by the application, implemented by adapters.
type EventDispatcher interface {
	// Dispatch publishes one or more domain events
	Dispatch(ctx context.Context, events []domain.Event) error
}
