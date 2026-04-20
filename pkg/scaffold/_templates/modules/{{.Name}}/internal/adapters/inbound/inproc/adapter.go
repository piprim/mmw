package inproc

import (
	def{{.Name | lower}} "{{.ContractsPath}}/definitions/{{.Name}}"
	"{{.OrgPrefix}}/{{.Name}}/internal/application"
)

// Adapter wraps application.{{.Name | pascal}}Service and implements def{{.Name | lower}}.{{.Name | pascal}}Service.
type Adapter struct {
	svc application.{{.Name | pascal}}Service
}

// compile-time assertion
var _ def{{.Name | lower}}.{{.Name | pascal}}Service = (*Adapter)(nil)

// NewAdapter creates a new Adapter.
func NewAdapter(svc application.{{.Name | pascal}}Service) *Adapter {
	return &Adapter{svc: svc}
}
