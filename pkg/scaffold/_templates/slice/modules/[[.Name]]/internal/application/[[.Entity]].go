[[if ne .Shape "gateway" -]]
package application

import (
	"context"
	"fmt"

[[if eq .Shape "aggregate"]]	"[[.ModulePath]]/internal/application/command"
[[end]]	"[[.ModulePath]]/internal/application/ports"
	"[[.ModulePath]]/internal/application/query"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

// [[.Entity | pascal]]Service is the primary port of the [[.Entity]] slice:
// implemented by [[.Entity | pascal]]AppService, called by adapters.
[[- if eq .Shape "read-model"]]
// It returns the read model directly: a DTO would only duplicate its fields.
[[- else]]
// It returns the aggregate directly: a DTO would only duplicate its getters.
[[- end]]
type [[.Entity | pascal]]Service interface {
	// Get[[.Entity | pascal]] retrieves a [[.Entity]] by its integer primary key.
	Get[[.Entity | pascal]](ctx context.Context, id int64) (*[[.Entity]].[[.Entity | pascal]], error)
[[- if eq .Shape "aggregate"]]
	// Delete[[.Entity | pascal]] deletes a [[.Entity]] and publishes its Deleted event.
	Delete[[.Entity | pascal]](ctx context.Context, id int64) error
[[- end]]
	// Health checks the storage.
	Health(ctx context.Context) (any, error)
}

[[- if eq .Shape "read-model"]]

// [[.Entity | pascal]]AppService implements [[.Entity | pascal]]Service by delegating
// to one query.
type [[.Entity | pascal]]AppService struct {
	repository ports.[[.Entity | pascal]]Repository
	getQuery   *query.Get[[.Entity | pascal]]Query
}

// New[[.Entity | pascal]]AppService creates a [[.Entity | pascal]]AppService.
func New[[.Entity | pascal]]AppService(repository ports.[[.Entity | pascal]]Repository) [[.Entity | pascal]]Service {
	return &[[.Entity | pascal]]AppService{
		repository: repository,
		getQuery:   query.NewGet[[.Entity | pascal]]Query(repository),
	}
}
[[- else]]

// [[.Entity | pascal]]AppService implements [[.Entity | pascal]]Service by delegating
// to one query and one command.
type [[.Entity | pascal]]AppService struct {
	repository ports.[[.Entity | pascal]]Repository
	getQuery   *query.Get[[.Entity | pascal]]Query
	deleteCmd  *command.Delete[[.Entity | pascal]]Command
}

// New[[.Entity | pascal]]AppService creates a [[.Entity | pascal]]AppService. The repository
// and the dispatcher must share the unit of work so a command's writes land
// in one transaction.
func New[[.Entity | pascal]]AppService(
	repository ports.[[.Entity | pascal]]Repository,
	unitOfWork ports.UnitOfWork,
	eventDispatcher ports.EventDispatcher,
) [[.Entity | pascal]]Service {
	return &[[.Entity | pascal]]AppService{
		repository: repository,
		getQuery:   query.NewGet[[.Entity | pascal]]Query(repository),
		deleteCmd:  command.NewDelete[[.Entity | pascal]]Command(repository, unitOfWork, eventDispatcher),
	}
}
[[- end]]

// Get[[.Entity | pascal]] retrieves a [[.Entity]] by ID.
func (s *[[.Entity | pascal]]AppService) Get[[.Entity | pascal]](
	ctx context.Context, id int64,
) (*[[.Entity]].[[.Entity | pascal]], error) {
	result, err := s.getQuery.Execute(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get [[.Entity]]: %w", err)
	}

	return result, nil
}

[[- if eq .Shape "aggregate"]]

// Delete[[.Entity | pascal]] deletes a [[.Entity]].
func (s *[[.Entity | pascal]]AppService) Delete[[.Entity | pascal]](ctx context.Context, id int64) error {
	if err := s.deleteCmd.Execute(ctx, id); err != nil {
		return fmt.Errorf("delete [[.Entity]]: %w", err)
	}

	return nil
}
[[- end]]

// Health returns a simple storage health check.
func (s *[[.Entity | pascal]]AppService) Health(ctx context.Context) (any, error) {
	count, err := s.repository.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("health check: %w", err)
	}

	return count, nil
}
[[- else -]]
package application

import (
	"context"
	"fmt"

	"[[.ModulePath]]/internal/application/command"
	"[[.ModulePath]]/internal/application/ports"
)

// [[.Entity | pascal]]Service is the primary port of the [[.Entity]] slice:
// implemented by [[.Entity | pascal]]AppService, called by adapters.
type [[.Entity | pascal]]Service interface {
	// Send[[.Entity | pascal]] validates the label, sends it through the gateway and publishes the Sent event.
	Send[[.Entity | pascal]](ctx context.Context, label string) error
	// Health checks the gateway.
	Health(ctx context.Context) (any, error)
}

// [[.Entity | pascal]]AppService implements [[.Entity | pascal]]Service by delegating
// to one command.
type [[.Entity | pascal]]AppService struct {
	gateway ports.[[.Entity | pascal]]Gateway
	sendCmd *command.Send[[.Entity | pascal]]Command
}

// New[[.Entity | pascal]]AppService creates a [[.Entity | pascal]]AppService.
func New[[.Entity | pascal]]AppService(
	gateway ports.[[.Entity | pascal]]Gateway, publisher ports.EventPublisher,
) [[.Entity | pascal]]Service {
	return &[[.Entity | pascal]]AppService{
		gateway: gateway,
		sendCmd: command.NewSend[[.Entity | pascal]]Command(gateway, publisher),
	}
}

// Send[[.Entity | pascal]] sends a [[.Entity]].
func (s *[[.Entity | pascal]]AppService) Send[[.Entity | pascal]](ctx context.Context, label string) error {
	if err := s.sendCmd.Execute(ctx, label); err != nil {
		return fmt.Errorf("send [[.Entity]]: %w", err)
	}

	return nil
}

// Health returns the gateway health check.
func (s *[[.Entity | pascal]]AppService) Health(ctx context.Context) (any, error) {
	status, err := s.gateway.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("health check: %w", err)
	}

	return status, nil
}
[[- end]]
