package mapper

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

// [[.Entity | pascal]]ToProto flattens the [[if eq .Shape "read-model"]]read model[[else]]aggregate[[end]] to its wire representation.
func [[.Entity | pascal]]ToProto(a *[[.Entity]].[[.Entity | pascal]]) *[[.Name]]v1.[[.Entity | pascal]] {
	return &[[.Name]]v1.[[.Entity | pascal]]{
[[- if eq .Shape "read-model"]]
		Id:        a.ID.Int64(),
		Label:     a.Label.String(),
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
[[- else]]
		Id:        a.ID().Int64(),
		Label:     a.Label().String(),
		CreatedAt: timestamppb.New(a.CreatedAt()),
		UpdatedAt: timestamppb.New(a.UpdatedAt()),
[[- end]]
	}
}
