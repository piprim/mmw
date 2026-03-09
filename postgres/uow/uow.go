package uow

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBExecutor abstracts both *pgxpool.Pool and pgx.Tx
// Every service's repository will use this.
type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row

	// SendBatch to support bulk operations safely
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type txKey struct{}

// GetExecutor safely extracts the transaction from the context, or falls back to the pool.
// Repositories call this directly.
func GetExecutor(ctx context.Context, pool *pgxpool.Pool) DBExecutor {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}

	return pool
}

// UnitOfWork is the generic Postgres implementation
type UnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

func (u *UnitOfWork) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	// 1. Check if we are ALREADY inside a transaction
	if existingTx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		// We are nested! Tell pgx to create a SAVEPOINT instead of a new connection.
		nestedTx, err := existingTx.Begin(ctx)
		if err != nil {
			return fmt.Errorf("cannot start nested transaction (savepoint): %w", err)
		}

		// This will only rollback to the SAVEPOINT, leaving the outer transaction intact
		defer func() { _ = nestedTx.Rollback(ctx) }()

		// Wrap the nestedTx in the context for any deeper calls
		ctxWithNestedTx := context.WithValue(ctx, txKey{}, nestedTx)

		if err := fn(ctxWithNestedTx); err != nil {
			return err
		}

		// "Committing" a savepoint just releases it, it doesn't commit the outer transaction
		if err := nestedTx.Commit(ctx); err != nil {
			return fmt.Errorf("cannot release savepoint: %w", err)
		}

		return nil
	}

	// 2. We are NOT nested. Start a brand new root transaction from the pool.
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cannot start root transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ctxWithTx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(ctxWithTx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("cannot commit root transaction: %w", err)
	}

	return nil
}
