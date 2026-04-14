package {{.PkgDef}}

import (
	"context"
	"fmt"
)

// InprocClient is a thin wrapper that accepts any {{.NameTitle}}Service implementation.
type InprocClient struct {
	server {{.NameTitle}}Service
}

func NewInprocClient(server {{.NameTitle}}Service) *InprocClient {
	return &InprocClient{server: server}
}

func (c *InprocClient) Health(ctx context.Context) (any, error) {
	resp, err := c.server.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return resp, nil
}

// compile-time assertion
var _ {{.NameTitle}}Service = (*InprocClient)(nil)
