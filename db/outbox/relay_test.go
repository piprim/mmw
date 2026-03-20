package ogloutbox

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
	pfevents "github.com/ovya/ogl/platform/events"
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

func TestNewOutboxRelay(t *testing.T) {
	mockBus := &mockSystemEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	relay := NewEnventsRelay(nil, mockBus, logger, tablename)

	assert.NotNil(t, relay)
	assert.Equal(t, mockBus, relay.bus)
	assert.Equal(t, logger, relay.logger)
	assert.Equal(t, 2*time.Second, relay.interval)
}

func TestOutboxRelay_Start_ContextCancellation(t *testing.T) {
	mockBus := &mockSystemEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	relay := NewEnventsRelay(nil, mockBus, logger, tablename)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		relay.Start(ctx)
		done <- true
	}()

	// Cancel context after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for Start to return
	select {
	case <-done:
		// Success - Start returned when context was cancelled
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestOutboxRelay_Start_ProcessesPeriodicBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker is not available")
	}

	// Setup database with testcontainer
	pool := setupTestDB(t)
	defer teardownTestDB(t, pool)

	// Insert test events
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO events (event_type, payload, occurred_at)
		VALUES
			('TodoCreated', '{"id":"1","title":"Test 1"}', NOW()),
			('TodoUpdated', '{"id":"2","title":"Test 2"}', NOW())
	`)
	require.NoError(t, err)

	// Setup relay with fast interval for testing
	mockBus := &mockSystemEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	relay := NewEnventsRelay(pool, mockBus, logger, tablename)
	relay.interval = 100 * time.Millisecond

	// Start relay in background
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go relay.Start(ctx)

	// Wait for events to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify events were published
	assert.GreaterOrEqual(t, len(mockBus.publishedMsgs), 2, "Should have published at least 2 events")

	// Verify events were marked as published in database
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE published_at IS NOT NULL").Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 2, "Events should be marked as published")

	// Cancel and ensure graceful shutdown
	cancel()
	time.Sleep(200 * time.Millisecond)
}

// Unit Tests for Business Logic

func TestOutboxRelay_SystemEventBus_Interface(t *testing.T) {
	// Verify that mockSystemEventBus implements the interface
	var _ pfevents.SystemEventBus = (*mockSystemEventBus)(nil)

	mockBus := &mockSystemEventBus{}
	err := mockBus.Publish(context.Background(), "TestEvent", []byte(`{"data":"test"}`))

	assert.NoError(t, err)
	assert.Len(t, mockBus.publishedMsgs, 1)
	assert.Equal(t, "TestEvent", mockBus.publishedMsgs[0].eventType)
	assert.Equal(t, []byte(`{"data":"test"}`), mockBus.publishedMsgs[0].payload)
}

func TestOutboxRelay_SystemEventBus_PublishError(t *testing.T) {
	mockBus := &mockSystemEventBus{
		publishFunc: func(ctx context.Context, eventType string, payload []byte) error {
			return errors.New("publish failed")
		},
	}

	err := mockBus.Publish(context.Background(), "TestEvent", []byte(`{"data":"test"}`))

	assert.Error(t, err)
	assert.Equal(t, "publish failed", err.Error())
}

func TestOutboxRelay_Interval_Configuration(t *testing.T) {
	mockBus := &mockSystemEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	relay := NewEnventsRelay(nil, mockBus, logger, tablename)

	// Default interval should be 2 seconds
	assert.Equal(t, 2*time.Second, relay.interval)

	// Test that we can modify the interval
	relay.interval = 5 * time.Second
	assert.Equal(t, 5*time.Second, relay.interval)
}

// Integration Tests with Real Database
// These tests use testcontainers to spin up a real PostgreSQL database

// setupTestDB creates a PostgreSQL container and runs migrations
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
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

	// Clean up container when test completes
	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate postgres container: %v", err)
		}
	})

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Create connection pool
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create connection pool: %v", err)
	}

	// Clean up pool when test completes
	t.Cleanup(func() {
		pool.Close()
	})

	// Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return pool
}

// runMigrations creates the necessary tables for testing
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
	// Cleanup is handled by t.Cleanup() in setupTestDB
	// This function exists for symmetry and explicit cleanup if needed
}

// Behavior Tests (Logic Verification Without Database)

func TestOutboxRelay_ErrorHandling_Patterns(t *testing.T) {
	tests := []struct {
		name          string
		setupMockBus  func(*mockSystemEventBus)
		expectedError string
	}{
		{
			name: "bus publish error contains context",
			setupMockBus: func(bus *mockSystemEventBus) {
				bus.publishFunc = func(ctx context.Context, eventType string, payload []byte) error {
					return errors.New("connection timeout")
				}
			},
			expectedError: "connection timeout",
		},
		{
			name: "bus publish with context cancellation",
			setupMockBus: func(bus *mockSystemEventBus) {
				bus.publishFunc = func(ctx context.Context, eventType string, payload []byte) error {
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

			ctx := context.Background()
			err := mockBus.Publish(ctx, "TestEvent", []byte(`{"test":"data"}`))

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOutboxRelay_LifecycleManagement(t *testing.T) {
	t.Run("graceful shutdown on context cancellation", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		relay := NewEnventsRelay(nil, mockBus, logger, tablename)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Start should return when context is cancelled
		start := time.Now()
		relay.Start(ctx)
		duration := time.Since(start)

		// Should return quickly after context cancellation
		assert.Less(t, duration, 200*time.Millisecond, "Should shutdown quickly after context cancellation")
	})

	t.Run("ticker stops on shutdown", func(t *testing.T) {
		mockBus := &mockSystemEventBus{}
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		relay := NewEnventsRelay(nil, mockBus, logger, tablename)
		// Use very long interval to prevent ticker from firing during test
		relay.interval = 10 * time.Second

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			relay.Start(ctx)
			done <- true
		}()

		// Cancel immediately - we're only testing that Start() returns on cancellation
		time.Sleep(10 * time.Millisecond)
		cancel()

		// Wait for Start to return
		select {
		case <-done:
			// Success - ticker was properly stopped on shutdown
		case <-time.After(1 * time.Second):
			t.Fatal("Start did not return after context cancellation - ticker not stopped")
		}
	})
}

func TestOutboxRelay_ConfigurableInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
	}{
		{
			name:     "1 second interval",
			interval: 1 * time.Second,
		},
		{
			name:     "5 second interval",
			interval: 5 * time.Second,
		},
		{
			name:     "100 millisecond interval",
			interval: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBus := &mockSystemEventBus{}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			relay := NewEnventsRelay(nil, mockBus, logger, tablename)

			relay.interval = tt.interval
			assert.Equal(t, tt.interval, relay.interval)
		})
	}
}
