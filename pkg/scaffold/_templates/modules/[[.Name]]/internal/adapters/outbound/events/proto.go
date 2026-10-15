package events

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// domainToProto maps a domain event to its proto message, the payload other
// modules decode. One case per event; a nil result falls back to JSON.
func domainToProto(event domain.Event) protoreflect.ProtoMessage {
	switch e := event.(type) { //nolint:gocritic,revive // single-case switch: one case per event, more come with new events
[[- if .Database]]
	case *[[.Entity]].Deleted:
		// Assigned field by field: a literal would align its keys on the
		// entity name, which gofmt re-flows for every other name length.
		ev := &[[.Name]]v1.[[.Entity | pascal]]DeletedEvent{}
		ev.[[.Entity | pascal]]Id = e.GetAggregateID()
		ev.UserId = e.GetUserID().String()
		ev.DeletedAt = timestamppb.New(e.GetOccurredAt())

		return ev
[[- else]]
	case *[[.Entity]].Sent:
		return &[[.Name]]v1.[[.Entity | pascal]]SentEvent{
			Label:  e.Label,
			UserId: e.GetUserID().String(),
			SentAt: timestamppb.New(e.GetOccurredAt()),
		}
[[- end]]
	}

	return nil
}
