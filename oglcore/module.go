package oglcore

import (
	"context"
	"net/http"
)

// Module defines the contract for all isolated domains in the monolith
type Module interface {
	// RegisterRoutes allows the module to attach its HTTP/gRPC handlers to the router
	RegisterRoutes(mux *http.ServeMux)

	// StartWorkers runs background tasks (Outbox Relays, Event Listeners).
	// It should block until the context is canceled or a fatal error occurs.
	StartWorkers(ctx context.Context) error

	// Guaranteed to be called when the application is shutting down
	Close() error
}
