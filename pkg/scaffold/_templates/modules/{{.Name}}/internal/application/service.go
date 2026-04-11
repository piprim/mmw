package application

import "context"

// {{.NameTitle}}Service defines the application service interface.
type {{.NameTitle}}Service interface {
	Health(ctx context.Context) (any, error)
}

// {{.NameTitle}}ApplicationService implements {{.NameTitle}}Service.
type {{.NameTitle}}ApplicationService struct{}

// New{{.NameTitle}}ApplicationService creates a new application service.
func New{{.NameTitle}}ApplicationService(
	repo interface{},
	uow interface{},
	dispatcher interface{},
) {{.NameTitle}}Service {
	return &{{.NameTitle}}ApplicationService{}
}

func (s *{{.NameTitle}}ApplicationService) Health(ctx context.Context) (any, error) {
	return nil, nil
}
