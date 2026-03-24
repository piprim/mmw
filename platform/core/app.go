package oglcore

import (
	"context"
)

// Module defines the contract for all isolated applications in the monolith
type Module interface {
	// Start runs background tasks (Outbox Relays, Event Listeners).
	// It should block until the context is canceled or a fatal error occurs.
	Start(ctx context.Context) error
}
