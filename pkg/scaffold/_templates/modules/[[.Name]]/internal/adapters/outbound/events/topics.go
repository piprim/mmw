package events

import (
	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// domainTopics maps semantic domain event types to Watermill routing keys:
// the single place where domain semantics become transport concerns. The
// keys come from the (options.v1.topic) option of the proto event messages.
var domainTopics = map[string]string{
[[- if .Database]]
	[[.Entity]].EventTypeDeleted: [[.Name]]def.Topic[[.Entity | pascal]]Deleted,
[[- else]]
	[[.Entity]].EventTypeSent: [[.Name]]def.Topic[[.Entity | pascal]]Sent,
[[- end]]
}
