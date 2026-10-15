package ports

import (
	"context"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// [[.Entity | pascal]]Gateway is the secondary port to the external service that
// delivers a [[.Entity]]: needed by the application, implemented by adapters.
type [[.Entity | pascal]]Gateway interface {
	// [[.Entity | pascal]]Send hands the message to the provider.
	// Returns [[.Entity]].ErrGatewayUnavailable when the provider cannot be reached or refuses it.
	[[.Entity | pascal]]Send(ctx context.Context, a *[[.Entity]].[[.Entity | pascal]]) error

	// Health reports whether the provider answers.
	Health(ctx context.Context) (any, error)
}
