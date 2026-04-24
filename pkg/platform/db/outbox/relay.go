package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/piprim/mmw/pkg/platform/core"
	pfevents "github.com/piprim/mmw/pkg/platform/events"
	"github.com/rotisserie/eris"
)

const tableNameRegex = "(^[a-z_0-9]+\\.?[a-z_0-9]+$)|(^[a-z_0-9]+$)"

// EventsRelay coordinates the reliable transfer of events from a database outbox
// table to an external messaging system (SystemEventBus).
type EventsRelay struct {
	pool      *pgxpool.Pool           // The database connection pool.
	bus       pfevents.SystemEventBus // The transport mechanism (NATS, RabbitMQ, etc.).
	logger    *slog.Logger            // Structured logger for operational monitoring.
	interval  time.Duration           // Frequency at which the relay polls the database.
	tableName string                  // The specific SQL table to poll (e.g., "todo_events").
}

func NewEventsRelay(
	pool *pgxpool.Pool,
	bus pfevents.SystemEventBus,
	logger *slog.Logger,
	tableName string,
) (*EventsRelay, error) {
	matched, err := regexp.MatchString(tableNameRegex, tableName)
	if err != nil {
		return nil, fmt.Errorf(`failed to compile regexp: %w`, err)
	}

	if !matched {
		return nil, fmt.Errorf(`table name "%s" is not allowed`, tableName)
	}

	return &EventsRelay{
		pool:      pool,
		bus:       bus,
		logger:    logger,
		interval:  2 * time.Second, // Poll every 2 seconds
		tableName: tableName,
	}, nil
}

// Start polls the outbox table on a fixed interval, publishing pending events to the bus.
// It blocks until ctx is cancelled; transient failures are logged and retried on the next tick.
// Use AsModule to integrate with platform.New.
func (r *EventsRelay) Start(ctx context.Context) {
	r.logger.Info("starting outbox relay worker")
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("shutting down outbox relay worker")

			return
		case <-ticker.C:
			if err := r.processBatch(ctx); err != nil {
				r.logger.Error("outbox processing failed", "error", err)
			}
		}
	}
}

// moduleFunc adapts a blocking func(context.Context) to core.Module.
type moduleFunc func(context.Context) error

func (f moduleFunc) Start(ctx context.Context) error { return f(ctx) }

// AsModule returns a core.Module adapter so the relay can be passed to platform.New.
// The module's Start blocks until ctx is cancelled and always returns nil.
func (r *EventsRelay) AsModule() core.Module {
	return moduleFunc(func(ctx context.Context) error {
		r.Start(ctx)
		return nil
	})
}

func (r *EventsRelay) processBatch(ctx context.Context) error {
	// 1. Open a transaction for the worker
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return eris.Wrap(err, "opening worker transaction failed")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 2. Fetch unpublished events (Lock them so other workers ignore them)
	query := fmt.Sprintf(`
		SELECT id, event_type, payload
		FROM %s
		WHERE published_at IS NULL
		ORDER BY occurred_at ASC
		LIMIT 100
		FOR UPDATE SKIP LOCKED
	`, r.tableName)
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return eris.Wrap(err, "fetching unpublished events failed")
	}
	defer rows.Close()

	var eventIDs []int

	for rows.Next() {
		var id int
		var eventType string
		var payload []byte

		if err := rows.Scan(&id, &eventType, &payload); err != nil {
			return eris.Wrap(err, "scaning unpublished events failed")
		}

		// 3. Publish to the real system bus
		if err := r.bus.Publish(ctx, eventType, payload); err != nil {
			// If publishing fails, we return. The defer tx.Rollback() unlocks the rows
			// so they can be retried on the next tick!
			return eris.Wrap(err, "publishing evants fails")
		}

		eventIDs = append(eventIDs, id)
	}
	rows.Close() // Must close rows before executing the next query

	// 4. If we published anything, mark them as done using pgx.Batch
	if len(eventIDs) > 0 {
		updateQuery := fmt.Sprintf(`
			UPDATE %s
			SET published_at = NOW()
			WHERE id = ANY($1)`, r.tableName)

		_, err := tx.Exec(ctx, updateQuery, eventIDs)
		if err != nil {
			return eris.Wrap(err, "mark published events as done failed")
		}

		r.logger.Info("processed outbox batch", "count", len(eventIDs))
	}

	return eris.Wrap(tx.Commit(ctx), "committing event worker failed")
}
