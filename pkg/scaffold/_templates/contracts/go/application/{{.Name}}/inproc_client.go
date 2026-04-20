package def{{.Name | lower}}

// InprocClient is a thin wrapper that accepts any {{.Name | pascal}}Service implementation.
type InprocClient struct {
	server {{.Name | pascal}}Service
}

func NewInprocClient(server {{.Name | pascal}}Service) *InprocClient {
	return &InprocClient{server: server}
}

var _ {{.Name | pascal}}Service = (*InprocClient)(nil)
