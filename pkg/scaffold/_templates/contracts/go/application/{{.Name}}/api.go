package {{.PkgDef}}

import "context"

// {{.NameTitle}}Service defines the public API contract for the {{.Name}} module.
type {{.NameTitle}}Service interface {
	// TODO: add methods matching your proto service definition
	Health(ctx context.Context) (any, error)
}
