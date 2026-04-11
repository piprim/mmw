package postgres

import (
	"context"

	"{{.ModulePath}}/internal/application/ports"
	"{{.ModulePath}}/internal/domain"
)

// {{.NameTitle}}Repository implements ports.{{.NameTitle}}Repository using PostgreSQL.
type {{.NameTitle}}Repository struct{}

// New{{.NameTitle}}Repository creates a new PostgreSQL repository.
func New{{.NameTitle}}Repository(uow interface{}) *{{.NameTitle}}Repository {
	return &{{.NameTitle}}Repository{}
}

var _ ports.{{.NameTitle}}Repository = (*{{.NameTitle}}Repository)(nil)

func (r *{{.NameTitle}}Repository) Save(ctx context.Context, e *domain.{{.NameTitle}}) error { return nil }
func (r *{{.NameTitle}}Repository) FindByID(ctx context.Context, id domain.{{.NameTitle}}ID) (*domain.{{.NameTitle}}, error) {
	return nil, domain.Err{{.NameTitle}}NotFound
}
func (r *{{.NameTitle}}Repository) Delete(ctx context.Context, id domain.{{.NameTitle}}ID) error { return nil }
func (r *{{.NameTitle}}Repository) Health(ctx context.Context) error                              { return nil }
