package inproc

import (
	"context"

	"github.com/rotisserie/eris"

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
	[[.Name]]v1 "[[.ContractsPath]]/go/network/[[.Name]]/v1"

	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/mapper"
	"[[.OrgPrefix]]/[[.Name]]/internal/application"
)

// [[.Entity | pascal]]ContractAdapter wraps application.[[.Entity | pascal]]Service and implements
// [[.Name]]def.[[.Entity | pascal]]Service, so other modules call this module in-process
// without depending on its internal packages.
type [[.Entity | pascal]]ContractAdapter struct {
	svc application.[[.Entity | pascal]]Service
}

var _ [[.Name]]def.[[.Entity | pascal]]Service = (*[[.Entity | pascal]]ContractAdapter)(nil)

// New[[.Entity | pascal]]ContractAdapter creates the adapter.
func New[[.Entity | pascal]]ContractAdapter(svc application.[[.Entity | pascal]]Service) *[[.Entity | pascal]]ContractAdapter {
	return &[[.Entity | pascal]]ContractAdapter{svc: svc}
}

[[- if .Database]]

func (a *[[.Entity | pascal]]ContractAdapter) Get[[.Entity | pascal]](
	ctx context.Context, req *[[.Name]]v1.Get[[.Entity | pascal]]Request,
) (*[[.Name]]v1.Get[[.Entity | pascal]]Response, error) {
	result, err := a.svc.Get[[.Entity | pascal]](ctx, req.GetId())
	if err != nil {
		return nil, eris.Wrap(mapper.DomainErrorFor(err), "get [[.Entity]]")
	}

	return &[[.Name]]v1.Get[[.Entity | pascal]]Response{[[.Entity | pascal]]: mapper.[[.Entity | pascal]]ToProto(result)}, nil
}

func (a *[[.Entity | pascal]]ContractAdapter) Delete[[.Entity | pascal]](
	ctx context.Context, req *[[.Name]]v1.Delete[[.Entity | pascal]]Request,
) (*[[.Name]]v1.Delete[[.Entity | pascal]]Response, error) {
	if err := a.svc.Delete[[.Entity | pascal]](ctx, req.GetId()); err != nil {
		return nil, eris.Wrap(mapper.DomainErrorFor(err), "delete [[.Entity]]")
	}

	return &[[.Name]]v1.Delete[[.Entity | pascal]]Response{}, nil
}
[[- else]]

func (a *[[.Entity | pascal]]ContractAdapter) Send[[.Entity | pascal]](
	ctx context.Context, req *[[.Name]]v1.Send[[.Entity | pascal]]Request,
) (*[[.Name]]v1.Send[[.Entity | pascal]]Response, error) {
	if err := a.svc.Send[[.Entity | pascal]](ctx, req.GetLabel()); err != nil {
		return nil, eris.Wrap(mapper.DomainErrorFor(err), "send [[.Entity]]")
	}

	return &[[.Name]]v1.Send[[.Entity | pascal]]Response{}, nil
}
[[- end]]
