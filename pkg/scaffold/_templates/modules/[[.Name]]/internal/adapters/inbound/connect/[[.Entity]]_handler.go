[[if .Database -]]
package connect

import (
	"context"

	"connectrpc.com/connect"

	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/mapper"
	"[[.OrgPrefix]]/[[.Name]]/internal/application"
)

// [[.Entity | pascal]]Handler implements the Connect [[.Entity | pascal]]ServiceHandler interface.
type [[.Entity | pascal]]Handler struct {
	service application.[[.Entity | pascal]]Service
}

// New[[.Entity | pascal]]Handler creates a new [[.Entity | pascal]]Handler.
func New[[.Entity | pascal]]Handler(service application.[[.Entity | pascal]]Service) *[[.Entity | pascal]]Handler {
	return &[[.Entity | pascal]]Handler{service: service}
}

// Get[[.Entity | pascal]] retrieves a [[.Entity]] by its integer primary key.
func (h *[[.Entity | pascal]]Handler) Get[[.Entity | pascal]](
	ctx context.Context,
	req *[[printf "connect.Request[%sv1.Get%sRequest]" .Name (pascal .Entity)]],
) (*[[printf "connect.Response[%sv1.Get%sResponse]" .Name (pascal .Entity)]], error) {
	a, err := h.service.Get[[.Entity | pascal]](ctx, req.Msg.GetId())
	if err != nil {
		return nil, connectErrorFrom(err)
	}

	return connect.NewResponse(&[[.Name]]v1.Get[[.Entity | pascal]]Response{[[.Entity | pascal]]: mapper.[[.Entity | pascal]]ToProto(a)}), nil
}

// Delete[[.Entity | pascal]] deletes a [[.Entity]].
func (h *[[.Entity | pascal]]Handler) Delete[[.Entity | pascal]](
	ctx context.Context,
	req *[[printf "connect.Request[%sv1.Delete%sRequest]" .Name (pascal .Entity)]],
) (*[[printf "connect.Response[%sv1.Delete%sResponse]" .Name (pascal .Entity)]], error) {
	if err := h.service.Delete[[.Entity | pascal]](ctx, req.Msg.GetId()); err != nil {
		return nil, connectErrorFrom(err)
	}

	return connect.NewResponse(&[[.Name]]v1.Delete[[.Entity | pascal]]Response{}), nil
}
[[- else -]]
package connect

import (
	"context"

	"connectrpc.com/connect"

	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/application"
)

// [[.Entity | pascal]]Handler implements the Connect [[.Entity | pascal]]ServiceHandler interface.
type [[.Entity | pascal]]Handler struct {
	service application.[[.Entity | pascal]]Service
}

// New[[.Entity | pascal]]Handler creates a new [[.Entity | pascal]]Handler.
func New[[.Entity | pascal]]Handler(service application.[[.Entity | pascal]]Service) *[[.Entity | pascal]]Handler {
	return &[[.Entity | pascal]]Handler{service: service}
}

// Send[[.Entity | pascal]] sends a [[.Entity]] through the gateway.
func (h *[[.Entity | pascal]]Handler) Send[[.Entity | pascal]](
	ctx context.Context,
	req *[[printf "connect.Request[%sv1.Send%sRequest]" .Name (pascal .Entity)]],
) (*[[printf "connect.Response[%sv1.Send%sResponse]" .Name (pascal .Entity)]], error) {
	if err := h.service.Send[[.Entity | pascal]](ctx, req.Msg.GetLabel()); err != nil {
		return nil, connectErrorFrom(err)
	}

	return connect.NewResponse(&[[.Name]]v1.Send[[.Entity | pascal]]Response{}), nil
}
[[- end]]
