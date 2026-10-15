package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/piprim/boa"
	pfuow "[[.PlatformPath]]/pkg/platform/pg/uow"
	"github.com/pivaldi/presence/v2"
	"github.com/rotisserie/eris"
	"google.golang.org/protobuf/encoding/protojson"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/outbound/persistence/postgres/dao/boamodel"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// PostgresOutboxDispatcher writes domain events to the [[.Name]].event outbox
// through boa, on the executor of the unit of work: the events land in the
// same transaction as the state change that raised them.
type PostgresOutboxDispatcher struct {
	boa *boa.DB
}

// NewPostgresOutboxDispatcher creates the dispatcher. The boa DB has no pool
// of its own: every insert goes through the unit of work.
func NewPostgresOutboxDispatcher(uow *pfuow.UnitOfWork) *PostgresOutboxDispatcher {
	return &PostgresOutboxDispatcher{
		boa: boa.New(nil, boa.WithExecutorResolver(func(ctx context.Context) boa.DBExecutor {
			return uow.Executor(ctx)
		})),
	}
}

// Dispatch saves all events to the outbox table in a single multi-row INSERT.
// The stored event_type is the Watermill routing key, resolved via domainTopics.
// ponytail: one statement caps at 65535 bind parameters, i.e. 21845 events;
// chunk the slice if a command ever raises more.
func (d *PostgresOutboxDispatcher) Dispatch(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	models := make([]boamodel.[[.Name | pascal]]Event, len(events))

	for i, event := range events {
		topic, ok := domainTopics[event.EventType()]
		if !ok {
			return eris.Errorf("no routing key for domain event type %q", event.EventType())
		}

		var payload []byte
		var err error

		// Map domain events to proto messages where possible
		if protoMsg := domainToProto(event); protoMsg != nil {
			payload, err = protojson.Marshal(protoMsg)
		} else {
			payload, err = json.Marshal(event)
		}

		if err != nil {
			return eris.Wrapf(err, "failed to marshal event %s", event.EventType())
		}

		models[i] = boamodel.[[.Name | pascal]]Event{
			EventType:   topic,
			Payload:     payload,
			OccurredAt:  event.GetOccurredAt(),
			PublishedAt: presence.Null[time.Time](),
		}
	}

	if _, err := d.boa.NewInsert().Model(&models).Exec(ctx); err != nil {
		return eris.Wrap(err, "failed to insert outbox events")
	}

	return nil
}
