//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pfdbmigrator "[[.PlatformPath]]/pkg/platform/db/migrator"
	pfuow "[[.PlatformPath]]/pkg/platform/pg/uow"
	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/outbound/events"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
	"[[.OrgPrefix]]/[[.Name]]/internal/infra/persistence/migrations"
)

var [[.Entity]]TestUser = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// setupTestDB starts a Postgres container and applies the module migrations.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"), postgres.WithUsername("testuser"), postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	m, err := pfdbmigrator.New(db, migrations.FS, "scripts", "[[.Name]]")
	if err != nil {
		t.Fatalf("migrator: %v", err)
	}
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	return pool
}

func TestPostgres[[.Entity | pascal]]Repository(t *testing.T) {
	pool := setupTestDB(t)
	uow := pfuow.New(pool)
	repo := NewPostgres[[.Entity | pascal]]Repository(uow)
	ctx := context.Background()

	var seededID int64
	if err := pool.QueryRow(ctx, "INSERT INTO [[.Name]].[[.Entity]] (label) VALUES ('Première') RETURNING id").Scan(&seededID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Run("FindByID returns the row", func(t *testing.T) {
		a, err := repo.[[.Entity | pascal]]FindByID(ctx, [[.Entity]].ID(seededID))
		if err != nil {
			t.Fatalf("FindByID() error: %v", err)
		}
		if a.Label().String() != "Première" || a.CreatedAt().IsZero() {
			t.Errorf("got %q/%v", a.Label(), a.CreatedAt())
		}
	})

	t.Run("FindByID reports an unknown id as ErrNotFound", func(t *testing.T) {
		if _, err := repo.[[.Entity | pascal]]FindByID(ctx, [[.Entity]].ID(999999)); !errors.Is(err, [[.Entity]].ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})

	t.Run("Health counts the rows", func(t *testing.T) {
		got, err := repo.Health(ctx)
		if err != nil || got.(int) < 1 {
			t.Errorf("Health() = %v, %v", got, err)
		}
	})

	t.Run("Delete and Dispatch share the transaction", func(t *testing.T) {
		dispatcher := events.NewPostgresOutboxDispatcher(uow)
		errRollback := errors.New("rollback")
		err := uow.WithTransaction(ctx, func(txCtx context.Context) error {
			a, err := repo.[[.Entity | pascal]]FindByID(txCtx, [[.Entity]].ID(seededID))
			if err != nil {
				return err
			}
			a.Delete([[.Entity]]TestUser)
			if err := repo.[[.Entity | pascal]]Delete(txCtx, a.ID()); err != nil {
				return err
			}
			if err := dispatcher.Dispatch(txCtx, a.Events()); err != nil {
				return err
			}

			return errRollback
		})
		if !errors.Is(err, errRollback) {
			t.Fatalf("WithTransaction() error = %v, want errRollback", err)
		}
		if _, err := repo.[[.Entity | pascal]]FindByID(ctx, [[.Entity]].ID(seededID)); err != nil {
			t.Errorf("row gone after rollback: %v", err)
		}
		var n int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM [[.Name]].event").Scan(&n)
		if n != 0 {
			t.Errorf("outbox has %d rows after rollback, want 0", n)
		}
	})

	t.Run("Delete removes the row and the event lands in the outbox", func(t *testing.T) {
		dispatcher := events.NewPostgresOutboxDispatcher(uow)
		err := uow.WithTransaction(ctx, func(txCtx context.Context) error {
			a, err := repo.[[.Entity | pascal]]FindByID(txCtx, [[.Entity]].ID(seededID))
			if err != nil {
				return err
			}
			a.Delete([[.Entity]]TestUser)
			if err := repo.[[.Entity | pascal]]Delete(txCtx, a.ID()); err != nil {
				return err
			}

			return dispatcher.Dispatch(txCtx, a.Events())
		})
		if err != nil {
			t.Fatalf("WithTransaction() error: %v", err)
		}
		if _, err := repo.[[.Entity | pascal]]FindByID(ctx, [[.Entity]].ID(seededID)); !errors.Is(err, [[.Entity]].ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
		var eventType string
		var unpublished bool
		if err := pool.QueryRow(ctx, "SELECT event_type, published_at IS NULL FROM [[.Name]].event").Scan(&eventType, &unpublished); err != nil {
			t.Fatalf("outbox row: %v", err)
		}
		if eventType != "[[.Name]].[[.Entity]].deleted.v1" || !unpublished {
			t.Errorf("outbox row = %q/%v", eventType, unpublished)
		}
	})

	t.Run("Delete reports an unknown id as ErrNotFound", func(t *testing.T) {
		if err := repo.[[.Entity | pascal]]Delete(ctx, [[.Entity]].ID(999999)); !errors.Is(err, [[.Entity]].ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})
}
