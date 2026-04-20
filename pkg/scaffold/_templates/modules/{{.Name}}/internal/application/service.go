package application

import "context"

// {{.Name | pascal}}Service defines the application service interface.
type {{.Name | pascal}}Service interface {
	Health(ctx context.Context) (any, error)
}

// {{.Name | pascal}}ApplicationService implements {{.Name | pascal}}Service.
type {{.Name | pascal}}ApplicationService struct{}

// New{{.Name | pascal}}ApplicationService creates a new application service.
func New{{.Name | pascal}}ApplicationService(
	repo interface{},
	uow interface{},
	dispatcher interface{},
) {{.Name | pascal}}Service {
	return &{{.Name | pascal}}ApplicationService{}
}

func (s *{{.Name | pascal}}ApplicationService) Health(ctx context.Context) (any, error) {
	return nil, nil
}
