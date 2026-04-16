package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pfevents "github.com/piprim/mmw/pkg/platform/events"
	"github.com/rotisserie/eris"
)

// EventsRelay coordinates the reliable transfer of events from a database outbox
// table to an external messaging system (SystemEventBus).
type EventsRelay struct {
	pool      *pgxpool.Pool           // The database connection pool.
	bus       pfevents.SystemEventBus // The transport mechanism (NATS, RabbitMQ, etc.).
	logger    *slog.Logger            // Structured logger for operational monitoring.
	interval  time.Duration           // Frequency at which the relay polls the database.
	tableName string                  // The specific SQL table to poll (e.g., "todo_events").
}

func NewEnventsRelay(
	pool *pgxpool.Pool,
	bus pfevents.SystemEventBus,
	logger *slog.Logger,
	tableName string,
) *EventsRelay {
	return &EventsRelay{
		pool:      pool,
		bus:       bus,
		logger:    logger,
		interval:  2 * time.Second, // Poll every 2 seconds
		tableName: tableName,
	}
}

// Start runs continuously until the context is canceled (Graceful Shutdown)
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
