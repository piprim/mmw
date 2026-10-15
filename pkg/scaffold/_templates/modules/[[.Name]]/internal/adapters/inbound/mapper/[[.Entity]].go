package mapper

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

// [[.Entity | pascal]]ToProto flattens the aggregate to its wire representation.
func [[.Entity | pascal]]ToProto(a *[[.Entity]].[[.Entity | pascal]]) *[[.Name]]v1.[[.Entity | pascal]] {
	return &[[.Name]]v1.[[.Entity | pascal]]{
		Id:        a.ID().Int64(),
		Label:     a.Label().String(),
		CreatedAt: timestamppb.New(a.CreatedAt()),
		UpdatedAt: timestamppb.New(a.UpdatedAt()),
	}
}
