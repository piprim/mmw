package events

import (
	"testing"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

func TestDomainTopics(t *testing.T) {
[[- if .Database]]
	t.Run("maps the Deleted event to its topic", func(t *testing.T) {
		if got := domainTopics[[printf "[%s.EventTypeDeleted]" .Entity]]; got != [[.Name]]def.Topic[[.Entity | pascal]]Deleted {
			t.Errorf("domainTopics[Deleted] = %q, want %q", got, [[.Name]]def.Topic[[.Entity | pascal]]Deleted)
		}
	})
[[- else]]
	t.Run("maps the Sent event to its topic", func(t *testing.T) {
		if got := domainTopics[[printf "[%s.EventTypeSent]" .Entity]]; got != [[.Name]]def.Topic[[.Entity | pascal]]Sent {
			t.Errorf("domainTopics[Sent] = %q, want %q", got, [[.Name]]def.Topic[[.Entity | pascal]]Sent)
		}
	})
[[- end]]
	t.Run("a topic is never the domain event type", func(t *testing.T) {
		for domainType, topic := range domainTopics {
			if topic == domainType {
				t.Errorf("domainTopics[%q]: topic should be a routing key", domainType)
			}
		}
	})
}
