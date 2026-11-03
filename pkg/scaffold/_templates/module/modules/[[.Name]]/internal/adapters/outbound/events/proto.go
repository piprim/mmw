package events

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"[[.ModulePath]]/internal/domain"
)

// domainToProto maps a domain event to its proto message, the payload other
// modules decode. Each slice converts its own events (<entity>ToProto in
// <entity>_events.go); a nil result falls back to JSON.
func domainToProto(event domain.Event) protoreflect.ProtoMessage {
	_ = event // keeps revive's unused-parameter quiet while no slice converts events
	// mmw:slice-event
	return nil
}
