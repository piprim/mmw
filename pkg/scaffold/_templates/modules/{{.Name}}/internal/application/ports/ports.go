package ports

import (
	"context"

	"{{.OrgPrefix}}/{{.Name}}/internal/domain"
)

// {{.Name | pascal}}Repository defines the persistence port.
type {{.Name | pascal}}Repository interface {
	Save(ctx context.Context, e *domain.{{.Name | pascal}}) error
	FindByID(ctx context.Context, id domain.{{.Name | pascal}}ID) (*domain.{{.Name | pascal}}, error)
	Delete(ctx context.Context, id domain.{{.Name | pascal}}ID) error
	Health(ctx context.Context) error
}

// EventDispatcher dispatches domain events.
type EventDispatcher interface {
	Dispatch(ctx context.Context, events []domain.DomainEvent) error
}

// UnitOfWork wraps a database transaction.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
