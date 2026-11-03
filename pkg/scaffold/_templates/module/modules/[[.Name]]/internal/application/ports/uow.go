package ports

import "context"

// UnitOfWork defines the contract for atomic operations
type UnitOfWork interface {
	// WithTransaction executes fn in a transaction context allowing atomic operations.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
