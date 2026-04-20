package connect

import (
	"{{.OrgPrefix}}/{{.Name}}/internal/application"
)

// {{.Name | pascal}}Handler implements the Connect service handler.
type {{.Name | pascal}}Handler struct {
	service application.{{.Name | pascal}}Service
}

// New{{.Name | pascal}}Handler creates a new {{.Name | pascal}}Handler.
func New{{.Name | pascal}}Handler(service application.{{.Name | pascal}}Service) *{{.Name | pascal}}Handler {
	return &{{.Name | pascal}}Handler{service: service}
}
