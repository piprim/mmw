package events

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.ModulePath]]/internal/domain"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

// The [[.Entity]] event and its routing key, registered in domainTopics.
const (
[[- if eq .Shape "gateway"]]
	[[.Entity]]EventType = [[.Entity]].EventTypeSent
	[[.Entity]]Topic     = [[.Name]]def.Topic[[.Entity | pascal]]Sent
[[- else]]
	[[.Entity]]EventType = [[.Entity]].EventTypeDeleted
	[[.Entity]]Topic     = [[.Name]]def.Topic[[.Entity | pascal]]Deleted
[[- end]]
)

// [[.Entity]]ToProto maps a [[.Entity]] event to its proto message; nil for any
// other event. A second event of the slice turns this into a type switch.
func [[.Entity]]ToProto(event domain.Event) protoreflect.ProtoMessage {
[[- if ne .Shape "gateway"]]
	e, ok := event.(*[[.Entity]].Deleted)
	if !ok {
		return nil
	}

	// Assigned field by field: a literal would align its keys on the
	// entity name, which gofmt re-flows for every other name length.
	ev := &[[.Name]]v1.[[.Entity | pascal]]DeletedEvent{}
	ev.[[.Entity | pascal]]Id = e.GetAggregateID()
	ev.UserId = e.GetUserID().String()
	ev.DeletedAt = timestamppb.New(e.GetOccurredAt())

	return ev
[[- else]]
	e, ok := event.(*[[.Entity]].Sent)
	if !ok {
		return nil
	}

	return &[[.Name]]v1.[[.Entity | pascal]]SentEvent{
		Label:  e.Label,
		UserId: e.GetUserID().String(),
		SentAt: timestamppb.New(e.GetOccurredAt()),
	}
[[- end]]
}
