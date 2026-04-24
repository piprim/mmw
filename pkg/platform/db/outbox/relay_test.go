package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pfevents "github.com/piprim/mmw/pkg/platform/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const tablename = "events"

// mockSystemEventBus is a simple mock for testing
type mockSystemEventBus struct {
	publishFunc   func(ctx context.Context, eventType string, payload []byte) error
	publishedMsgs []publishedMessage
}

type publishedMessage struct {
	eventType string
	payload   []byte
}

func (m *mockSystemEventBus) Publish(ctx context.Context, eventType string, payload []byte) error {
	m.publishedMsgs = append(m.publishedMsgs, publishedMessage{
		eventType: eventType,
		payload:   payload,
	})

	if m.publishFunc != nil {
		return m.publishFunc(ctx, eventType, payload)
	}

	return nil
}

func TestNewEventsRelay(t *testing.T) {
	t.Run("creates relay with defaults", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		relay, err := NewEventsRelay(nil, mockBus, logger, tablename)

		require.Nil(t, err)
		assert.NotNil(t, relay)
		assert.Equal(t, mockBus, relay.bus)
		assert.Equal(t, logger, relay.logger)
		assert.Equal(t, 2*time.Second, relay.interval)
	})

	t.Run("rejects invalid table names", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		tableNames := []string{"", "\"", ".", "aze$rty", "a.b.c", "acc'ent"}

		for i := range tableNames {
			relay, err := NewEventsRelay(nil, mockBus, logger, tableNames[i])
			assert.Errorf(t, err, `table name "%s" should be rejected`, tableNames[i])
			assert.Nil(t, relay)
		}
	})

	t.Run("accepts valid table names", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		tableNames := []string{"a", "a.b", "aa", "aa.bb", "a2", "a2b", "a2b.b3c", "ab_c", "ab_c.d_e"}

		for i := range tableNames {
			relay, err := NewEventsRelay(nil, mockBus, logger, tableNames[i])
			assert.Nil(t, err, `table name "%s" should be accepted`, tableNames[i])
			assert.NotNil(t, relay)
		}
	})
}

func TestOutboxRelay_Start(t *testing.T) {
	t.Run("returns when context is cancelled", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		relay, err := NewEventsRelay(nil, mockBus, logger, tablename)
		require.Nil(t, err)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			relay.Start(ctx)
			done <- true
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Start did not return after context cancellation")
		}
	})

	t.Run("processes periodic batches", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}
		if _, err := exec.LookPath("docker"); err != nil {
			t.Skip("docker is not available")
		}

		pool := setupTestDB(t)
		defer teardownTestDB(t, pool)

		ctx := context.Background()
		_, err := pool.Exec(ctx, `
			INSERT INTO events (event_type, payload, occurred_at)
			VALUES
				('TodoCreated', '{"id":"1","title":"Test 1"}', NOW()),
				('TodoUpdated', '{"id":"2","title":"Test 2"}', NOW())
		`)
		require.NoError(t, err)

		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		relay, err := NewEventsRelay(pool, mockBus, logger, tablename)
		require.Nil(t, err)
		relay.interval = 100 * time.Millisecond

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		go relay.Start(ctx)

		time.Sleep(500 * time.Millisecond)

		assert.GreaterOrEqual(t, len(mockBus.publishedMsgs), 2)

		var count int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE published_at IS NOT NULL").Scan(&count)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 2)

		cancel()
		time.Sleep(200 * time.Millisecond)
	})
}

func TestOutboxRelay_SystemEventBus(t *testing.T) {
	t.Run("mock implements the interface", func(t *testing.T) {
		var _ pfevents.SystemEventBus = (*mockSystemEventBus)(nil)

		mockBus := &mockSystemEventBus{}
		err := mockBus.Publish(context.Background(), "TestEvent", []byte(`{"data":"test"}`))

		assert.NoError(t, err)
		assert.Len(t, mockBus.publishedMsgs, 1)
		assert.Equal(t, "TestEvent", mockBus.publishedMsgs[0].eventType)
		assert.Equal(t, []byte(`{"data":"test"}`), mockBus.publishedMsgs[0].payload)
	})

	t.Run("returns publish error from func", func(t *testing.T) {
		mockBus := &mockSystemEventBus{
			publishFunc: func(_ context.Context, _ string, _ []byte) error {
				return errors.New("publish failed")
			},
		}

		err := mockBus.Publish(context.Background(), "TestEvent", []byte(`{"data":"test"}`))

		assert.Error(t, err)
		assert.Equal(t, "publish failed", err.Error())
	})
}

func TestOutboxRelay_Interval(t *testing.T) {
	t.Run("defaults to 2 seconds", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		relay, err := NewEventsRelay(nil, mockBus, logger, tablename)
		require.Nil(t, err)
		assert.Equal(t, 2*time.Second, relay.interval)
	})

	tests := []struct {
		name     string
		interval time.Duration
	}{
		{"1 second", 1 * time.Second},
		{"5 seconds", 5 * time.Second},
		{"100 milliseconds", 100 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run("configurable to "+tt.name, func(t *testing.T) {
			mockBus := &mockSystemEventBus{}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			relay, _ := NewEventsRelay(nil, mockBus, logger, tablename)
			relay.interval = tt.interval
			assert.Equal(t, tt.interval, relay.interval)
		})
	}
}

func TestOutboxRelay_LifecycleManagement(t *testing.T) {
	t.Run("graceful shutdown on context cancellation", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		relay, _ := NewEventsRelay(nil, mockBus, logger, tablename)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		relay.Start(ctx)
		duration := time.Since(start)

		assert.Less(t, duration, 200*time.Millisecond)
	})

	t.Run("ticker stops on shutdown", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		relay, _ := NewEventsRelay(nil, mockBus, logger, tablename)
		relay.interval = 10 * time.Second

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			relay.Start(ctx)
			done <- true
		}()

		time.Sleep(10 * time.Millisecond)
		cancel()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Start did not return after context cancellation")
		}
	})
}

func TestOutboxRelay_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupMockBus  func(*mockSystemEventBus)
		expectedError string
	}{
		{
			name: "bus publish error contains context",
			setupMockBus: func(bus *mockSystemEventBus) {
				bus.publishFunc = func(_ context.Context, _ string, _ []byte) error {
					return errors.New("connection timeout")
				}
			},
			expectedError: "connection timeout",
		},
		{
			name: "bus publish with context cancellation",
			setupMockBus: func(bus *mockSystemEventBus) {
				bus.publishFunc = func(ctx context.Context, _ string, _ []byte) error {
					if ctx.Err() != nil {
						return ctx.Err()
					}

					return nil
				}
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBus := &mockSystemEventBus{}
			if tt.setupMockBus != nil {
				tt.setupMockBus(mockBus)
			}

			err := mockBus.Publish(context.Background(), "TestEvent", []byte(`{"test":"data"}`))

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// setupTestDB creates a PostgreSQL container and runs migrations
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	t.Cleanup(pool.Close)

	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return pool
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE events (
			id SERIAL PRIMARY KEY,
			event_type VARCHAR(255) NOT NULL,
			payload JSONB NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			published_at TIMESTAMPTZ
		);
	`)
	if err != nil {
		return fmt.Errorf("creating events table: %w", err)
	}

	return nil
}

func teardownTestDB(t *testing.T, _ *pgxpool.Pool) {
	t.Helper()
}
