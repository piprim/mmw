package connect

import (
	"{{.ModulePath}}/internal/application"
)

// {{.NameTitle}}Handler implements the Connect service handler.
type {{.NameTitle}}Handler struct {
	service application.{{.NameTitle}}Service
}

// New{{.NameTitle}}Handler creates a new {{.NameTitle}}Handler.
func New{{.NameTitle}}Handler(service application.{{.NameTitle}}Service) *{{.NameTitle}}Handler {
	return &{{.NameTitle}}Handler{service: service}
}
