package postgres

import (
	"context"

	"{{.OrgPrefix}}/{{.Name}}/internal/application/ports"
	"{{.OrgPrefix}}/{{.Name}}/internal/domain"
)

// {{.Name | pascal}}Repository implements ports.{{.Name | pascal}}Repository using PostgreSQL.
type {{.Name | pascal}}Repository struct{}

// New{{.Name | pascal}}Repository creates a new PostgreSQL repository.
func New{{.Name | pascal}}Repository(uow interface{}) *{{.Name | pascal}}Repository {
	return &{{.Name | pascal}}Repository{}
}

var _ ports.{{.Name | pascal}}Repository = (*{{.Name | pascal}}Repository)(nil)

func (r *{{.Name | pascal}}Repository) Save(ctx context.Context, e *domain.{{.Name | pascal}}) error { return nil }
func (r *{{.Name | pascal}}Repository) FindByID(ctx context.Context, id domain.{{.Name | pascal}}ID) (*domain.{{.Name | pascal}}, error) {
	return nil, domain.Err{{.Name | pascal}}NotFound
}
func (r *{{.Name | pascal}}Repository) Delete(ctx context.Context, id domain.{{.Name | pascal}}ID) error { return nil }
func (r *{{.Name | pascal}}Repository) Health(ctx context.Context) error                              { return nil }
