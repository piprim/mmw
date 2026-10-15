package events

import (
	"context"
	"encoding/json"

	"github.com/rotisserie/eris"
	"google.golang.org/protobuf/encoding/protojson"

	pfevents "[[.PlatformPath]]/pkg/platform/events"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// BusPublisher publishes domain events straight to the system event bus:
// without a database there is no outbox, so delivery is at-most-once.
type BusPublisher struct {
	bus pfevents.SystemEventBus
}

// NewBusPublisher creates the publisher.
func NewBusPublisher(bus pfevents.SystemEventBus) *BusPublisher {
	return &BusPublisher{bus: bus}
}

// Publish sends each event on its topic, resolved via domainTopics.
func (p *BusPublisher) Publish(ctx context.Context, events []domain.Event) error {
	for _, event := range events {
		topic, ok := domainTopics[event.EventType()]
		if !ok {
			return eris.Errorf("no routing key for domain event type %q", event.EventType())
		}

		var payload []byte
		var err error
		if protoMsg := domainToProto(event); protoMsg != nil {
			payload, err = protojson.Marshal(protoMsg)
		} else {
			payload, err = json.Marshal(event)
		}
		if err != nil {
			return eris.Wrapf(err, "failed to marshal event %s", event.EventType())
		}

		if err := p.bus.Publish(ctx, topic, payload); err != nil {
			return eris.Wrapf(err, "failed to publish event %s", event.EventType())
		}
	}

	return nil
}
