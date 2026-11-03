package events

import (
[[- if eq .Shape "gateway"]]
	"context"
[[- end]]
	"testing"
[[- if eq .Shape "gateway"]]
	"time"

	"github.com/google/uuid"
[[- end]]

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
[[- if eq .Shape "gateway"]]

	"[[.ModulePath]]/internal/domain"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
[[- end]]
)

func Test[[.Entity | pascal]]Events(t *testing.T) {
[[- if ne .Shape "gateway"]]
	t.Run("maps the Deleted event to its topic", func(t *testing.T) {
		if got := domainTopics[[printf "[%sEventType]" .Entity]]; got != [[.Name]]def.Topic[[.Entity | pascal]]Deleted {
			t.Errorf("domainTopics[Deleted] = %q, want %q", got, [[.Name]]def.Topic[[.Entity | pascal]]Deleted)
		}
	})
[[- else]]
	t.Run("maps the Sent event to its topic", func(t *testing.T) {
		if got := domainTopics[[printf "[%sEventType]" .Entity]]; got != [[.Name]]def.Topic[[.Entity | pascal]]Sent {
			t.Errorf("domainTopics[Sent] = %q, want %q", got, [[.Name]]def.Topic[[.Entity | pascal]]Sent)
		}
	})

	t.Run("publishes the proto payload on the mapped topic", func(t *testing.T) {
		sent := &[[.Entity]].Sent{
			BaseEvent: domain.BaseEvent{AggregateID: "Bonjour", UserID: uuid.New(), OccurredAt: time.Now()},
			Label:     "Bonjour",
		}
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
[[- end]]
}
