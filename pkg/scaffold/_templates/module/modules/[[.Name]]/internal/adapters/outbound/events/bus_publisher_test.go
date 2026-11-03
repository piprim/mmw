package events

import (
	"context"
	"testing"

	"[[.ModulePath]]/internal/domain"
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

func TestBusPublisher_Publish(t *testing.T) {
	t.Run("publishes the payload on the mapped topic", func(t *testing.T) {
		withStubTopic(t)
		bus := &recordingBus{}
		if err := NewBusPublisher(bus).Publish(context.Background(), []domain.Event{stubEvent{}}); err != nil {
			t.Fatalf("Publish() error: %v", err)
		}
		if len(bus.topics) != 1 || bus.topics[0] != "stub.topic.v1" {
			t.Fatalf("topics = %v, want [stub.topic.v1]", bus.topics)
		}
		if string(bus.payloads[0]) == "" || string(bus.payloads[0])[0] != '{' {
			t.Errorf("payload = %s", bus.payloads[0])
		}
	})

	t.Run("an event without a topic is an error and nothing is published", func(t *testing.T) {
		bus := &recordingBus{}
		if err := NewBusPublisher(bus).Publish(context.Background(), []domain.Event{stubEvent{}}); err == nil {
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
