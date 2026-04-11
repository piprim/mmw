package events

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	pfuow "{{.PlatformPath}}/pkg/platform/pg/uow"
	"{{.ModulePath}}/internal/domain"
	"github.com/rotisserie/eris"
)

// PostgresOutboxDispatcher saves domain events to the outbox table.
type PostgresOutboxDispatcher struct {
	uow *pfuow.UnitOfWork
}

// NewPostgresOutboxDispatcher creates a new dispatcher.
func NewPostgresOutboxDispatcher(uow *pfuow.UnitOfWork) *PostgresOutboxDispatcher {
	return &PostgresOutboxDispatcher{uow: uow}
}

// Dispatch saves all events to the outbox table.
func (d *PostgresOutboxDispatcher) Dispatch(ctx context.Context, events []domain.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `INSERT INTO {{.Name}}.event (event_type, payload, occurred_at) VALUES ($1, $2::jsonb, $3)`

	for _, event := range events {
		topic, ok := domainTopics[event.EventType()]
		if !ok {
			return eris.Errorf("no routing key for domain event type %q", event.EventType())
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return eris.Wrapf(err, "marshal event %s", event.EventType())
		}

		batch.Queue(query, topic, string(payload), event.GetOccurredAt())
	}

	br := d.uow.Executor(ctx).SendBatch(ctx, batch)
	defer br.Close()

	for range events {
		if _, err := br.Exec(); err != nil {
			return eris.Wrap(err, "insert outbox event")
		}
	}

	return nil
}
