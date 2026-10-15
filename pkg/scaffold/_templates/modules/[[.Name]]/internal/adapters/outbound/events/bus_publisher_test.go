package events

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

type recordingBus struct {
	topics   []string
	payloads [][]byte
}

func (b *recordingBus) Publish(_ context.Context, topic string, payload []byte) error {
	b.topics = append(b.topics, topic)
	b.payloads = append(b.payloads, payload)

	return nil
}

type unknownEvent struct{ domain.BaseEvent }

func (unknownEvent) EventType() string { return "[[.Entity]].unknown" }

func TestBusPublisher_Publish(t *testing.T) {
	sent := &[[.Entity]].Sent{
		BaseEvent: domain.BaseEvent{AggregateID: "Bonjour", UserID: uuid.New(), OccurredAt: time.Now()},
		Label:     "Bonjour",
	}

	t.Run("publishes the proto payload on the mapped topic", func(t *testing.T) {
		bus := &recordingBus{}
		if err := NewBusPublisher(bus).Publish(context.Background(), []domain.Event{sent}); err != nil {
			t.Fatalf("Publish() error: %v", err)
		}
		if len(bus.topics) != 1 {
			t.Fatalf("topics = %v, want one", bus.topics)
		}
		if bus.topics[0] != [[.Name]]def.Topic[[.Entity | pascal]]Sent {
			t.Errorf("topics = %v", bus.topics)
		}
		if string(bus.payloads[0]) == "" || string(bus.payloads[0])[0] != '{' {
			t.Errorf("payload = %s", bus.payloads[0])
		}
	})

	t.Run("an event without a topic is an error and nothing is published", func(t *testing.T) {
		bus := &recordingBus{}
		if err := NewBusPublisher(bus).Publish(context.Background(), []domain.Event{unknownEvent{}}); err == nil {
			t.Fatal("Publish() expected an error")
		}
		if len(bus.topics) != 0 {
			t.Errorf("published %d, want 0", len(bus.topics))
		}
	})

	t.Run("no events is a no-op", func(t *testing.T) {
		bus := &recordingBus{}
		if err := NewBusPublisher(bus).Publish(context.Background(), nil); err != nil || len(bus.topics) != 0 {
			t.Errorf("err/topics = %v/%v", err, bus.topics)
		}
	})
}
