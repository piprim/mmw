//go:build integration

package uow

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupIntegrationPool starts a PostgreSQL container and returns a ready pool.
// The container and pool are automatically cleaned up via t.Cleanup.
func setupIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not available")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	return pool
}

// createTestTable creates a minimal table for integration tests and drops it on cleanup.
func createTestTable(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_items (
			id   SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DROP TABLE IF EXISTS test_items")
	})
}

func TestUnitOfWork_Integration_Executor(t *testing.T) {
	t.Run("outside transaction returns pool executor", func(t *testing.T) {
		pool := setupIntegrationPool(t)
		ctx := context.Background()

		u := New(pool)
		executor := u.Executor(ctx)

		_, isTx := executor.(pgx.Tx)
		assert.False(t, isTx)

		var one int
		err := executor.QueryRow(ctx, "SELECT 1").Scan(&one)
		require.NoError(t, err)
		assert.Equal(t, 1, one)
	})

	t.Run("inside WithTransaction returns pgx.Tx executor", func(t *testing.T) {
		pool := setupIntegrationPool(t)
		ctx := context.Background()

		u := New(pool)

		err := u.WithTransaction(ctx, func(txCtx context.Context) error {
			executor := u.Executor(txCtx)

			_, isTx := executor.(pgx.Tx)
			assert.True(t, isTx)

			var one int

			return executor.QueryRow(txCtx, "SELECT 1").Scan(&one)
		})
		require.NoError(t, err)
	})
}

func TestUnitOfWork_Integration_WithTransaction(t *testing.T) {
	t.Run("commits when function returns nil", func(t *testing.T) {
		pool := setupIntegrationPool(t)
		ctx := context.Background()
		createTestTable(ctx, t, pool)

		u := New(pool)

		err := u.WithTransaction(ctx, func(txCtx context.Context) error {
			_, err := u.Executor(txCtx).Exec(txCtx,
				"INSERT INTO test_items (name) VALUES ($1)", "committed-row")

			return err
		})
		require.NoError(t, err)

		var name string
		err = pool.QueryRow(ctx, "SELECT name FROM test_items WHERE name = $1", "committed-row").Scan(&name)
		require.NoError(t, err)
		assert.Equal(t, "committed-row", name)
	})

	t.Run("rolls back when function returns error", func(t *testing.T) {
		pool := setupIntegrationPool(t)
		ctx := context.Background()
		createTestTable(ctx, t, pool)

		u := New(pool)

		forcedErr := errors.New("forced rollback")
		err := u.WithTransaction(ctx, func(txCtx context.Context) error {
			if _, execErr := u.Executor(txCtx).Exec(txCtx,
				"INSERT INTO test_items (name) VALUES ($1)", "rolled-back-row"); execErr != nil {
				return execErr
			}

			return forcedErr
		})
		require.ErrorIs(t, err, forcedErr)

		var count int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM test_items WHERE name = $1", "rolled-back-row").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestUnitOfWork_Integration_Nested(t *testing.T) {
	t.Run("both inner and outer commit", func(t *testing.T) {
		pool := setupIntegrationPool(t)
		ctx := context.Background()
		createTestTable(ctx, t, pool)

		u := New(pool)

		err := u.WithTransaction(ctx, func(outerCtx context.Context) error {
			if _, err := u.Executor(outerCtx).Exec(outerCtx,
				"INSERT INTO test_items (name) VALUES ($1)", "outer-row"); err != nil {
				return err
			}

			return u.WithTransaction(outerCtx, func(innerCtx context.Context) error {
				_, err := u.Executor(innerCtx).Exec(innerCtx,
					"INSERT INTO test_items (name) VALUES ($1)", "inner-row")

				return err
			})
		})
		require.NoError(t, err)

		var count int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM test_items WHERE name IN ('outer-row','inner-row')").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("inner rollback to savepoint while outer commits", func(t *testing.T) {
		pool := setupIntegrationPool(t)
		ctx := context.Background()
		createTestTable(ctx, t, pool)

		u := New(pool)

		err := u.WithTransaction(ctx, func(outerCtx context.Context) error {
			if _, err := u.Executor(outerCtx).Exec(outerCtx,
				"INSERT INTO test_items (name) VALUES ($1)", "outer-kept"); err != nil {
				return err
			}

			// Inner fn returns an error → savepoint is rolled back.
			// The outer transaction ignores the inner failure and commits anyway.
			_ = u.WithTransaction(outerCtx, func(innerCtx context.Context) error {
				if _, err := u.Executor(innerCtx).Exec(innerCtx,
					"INSERT INTO test_items (name) VALUES ($1)", "inner-discarded"); err != nil {
					return err
				}

				return errors.New("inner failure")
			})

			return nil
		})
		require.NoError(t, err)

		var outerCount int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM test_items WHERE name = 'outer-kept'").Scan(&outerCount)
		require.NoError(t, err)
		assert.Equal(t, 1, outerCount)

		var innerCount int
		err = pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM test_items WHERE name = 'inner-discarded'").Scan(&innerCount)
		require.NoError(t, err)
		assert.Equal(t, 0, innerCount)
	})

	t.Run("outer failure rolls back everything", func(t *testing.T) {
		pool := setupIntegrationPool(t)
		ctx := context.Background()
		createTestTable(ctx, t, pool)

		u := New(pool)

		outerErr := errors.New("outer failure")
		err := u.WithTransaction(ctx, func(outerCtx context.Context) error {
			if _, err := u.Executor(outerCtx).Exec(outerCtx,
				"INSERT INTO test_items (name) VALUES ($1)", "should-not-exist"); err != nil {
				return err
			}

			_ = u.WithTransaction(outerCtx, func(innerCtx context.Context) error {
				_, err := u.Executor(innerCtx).Exec(innerCtx,
					"INSERT INTO test_items (name) VALUES ($1)", "also-should-not-exist")

				return err
			})

			return outerErr
		})
		require.ErrorIs(t, err, outerErr)

		var count int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM test_items").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
