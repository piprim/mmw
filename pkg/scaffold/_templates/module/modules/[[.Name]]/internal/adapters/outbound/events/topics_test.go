package events

import (
	"testing"

	"[[.ModulePath]]/internal/domain"
)

// stubEvent is a domain event no slice owns; tests register its topic with
// withStubTopic.
type stubEvent struct{ domain.BaseEvent }

func (stubEvent) EventType() string { return "stub.event" }

// withStubTopic maps stubEvent to a topic for the duration of the test.
func withStubTopic(t *testing.T) {
	t.Helper()
	domainTopics["stub.event"] = "stub.topic.v1"
	t.Cleanup(func() { delete(domainTopics, "stub.event") })
}

func TestDomainTopics(t *testing.T) {
	t.Run("a topic is never the domain event type", func(t *testing.T) {
		withStubTopic(t)
		for domainType, topic := range domainTopics {
			if topic == domainType {
				t.Errorf("domainTopics[%q]: topic should be a routing key", domainType)
			}
		}
	})

	t.Run("an event without a proto mapping has no proto payload", func(t *testing.T) {
		if domainToProto(stubEvent{}) != nil {
			t.Error("domainToProto(stubEvent) should be nil")
		}
	})
}
