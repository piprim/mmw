package outbox

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not available")
	}
}

func TestProcessBatch_NoEvents(t *testing.T) {
	skipIfNoDocker(t)

	pool := setupTestDB(t)
	ctx := context.Background()

	mockBus := &mockSystemEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	relay := NewEnventsRelay(pool, mockBus, logger, tablename)

	err := relay.processBatch(ctx)

	require.NoError(t, err)
	assert.Empty(t, mockBus.publishedMsgs, "no messages should be published when table is empty")
}

func TestProcessBatch_PublishesAndMarksAsDone(t *testing.T) {
	skipIfNoDocker(t)

	pool := setupTestDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO events (event_type, payload, occurred_at)
		VALUES
			('OrderCreated', '{"id":"1"}', NOW()),
			('OrderUpdated', '{"id":"2"}', NOW())
	`)
	require.NoError(t, err)

	mockBus := &mockSystemEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	relay := NewEnventsRelay(pool, mockBus, logger, tablename)

	err = relay.processBatch(ctx)
	require.NoError(t, err)

	assert.Len(t, mockBus.publishedMsgs, 2)

	// Both events should be marked as published
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE published_at IS NOT NULL").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestProcessBatch_SkipsAlreadyPublished(t *testing.T) {
	skipIfNoDocker(t)

	pool := setupTestDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO events (event_type, payload, occurred_at, published_at)
		VALUES ('AlreadyDone', '{"id":"1"}', NOW(), NOW())
	`)
	require.NoError(t, err)

	mockBus := &mockSystemEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	relay := NewEnventsRelay(pool, mockBus, logger, tablename)

	err = relay.processBatch(ctx)
	require.NoError(t, err)

	assert.Empty(t, mockBus.publishedMsgs, "already-published events must be skipped")
}

func TestProcessBatch_RollsBackOnPublishError(t *testing.T) {
	skipIfNoDocker(t)

	pool := setupTestDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO events (event_type, payload, occurred_at)
		VALUES ('FailEvent', '{"id":"1"}', NOW())
	`)
	require.NoError(t, err)

	mockBus := &mockSystemEventBus{
		publishFunc: func(_ context.Context, _ string, _ []byte) error {
			return errors.New("bus unavailable")
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	relay := NewEnventsRelay(pool, mockBus, logger, tablename)

	err = relay.processBatch(ctx)
	assert.Error(t, err)

	// Transaction was rolled back: event must still be unpublished
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE published_at IS NULL").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "event should remain unpublished after a failed publish")
}

func TestProcessBatch_MixedPublishedAndUnpublished(t *testing.T) {
	skipIfNoDocker(t)

	pool := setupTestDB(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO events (event_type, payload, occurred_at, published_at)
		VALUES ('Done', '{"id":"1"}', NOW(), NOW());

		INSERT INTO events (event_type, payload, occurred_at)
		VALUES ('Pending', '{"id":"2"}', NOW());
	`)
	require.NoError(t, err)

	mockBus := &mockSystemEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	relay := NewEnventsRelay(pool, mockBus, logger, tablename)

	err = relay.processBatch(ctx)
	require.NoError(t, err)

	// Only the pending event should have been published
	assert.Len(t, mockBus.publishedMsgs, 1)
	assert.Equal(t, "Pending", mockBus.publishedMsgs[0].eventType)

	// Both events should now have published_at set
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM events WHERE published_at IS NOT NULL").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
