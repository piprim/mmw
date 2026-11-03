package [[.Name]]

import (
	[[if ne .Shape "gateway"]]pfuow "[[.PlatformPath]]/pkg/platform/pg/uow"

	[[end]][[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"

	"[[.ModulePath]]/internal/adapters/inbound/inproc"
[[- if ne .Shape "read-model"]]
	"[[.ModulePath]]/internal/adapters/outbound/events"
[[- end]]
[[- if eq .Shape "gateway"]]
	"[[.ModulePath]]/internal/adapters/outbound/gateway"
[[- else]]
	"[[.ModulePath]]/internal/adapters/outbound/persistence/postgres"
[[- end]]
	"[[.ModulePath]]/internal/application"
	"[[.ModulePath]]/internal/infra/config"
)
[[- if eq .Shape "gateway"]]

// new[[.Entity | pascal]]Service builds the HTTP gateway and the bus publisher.
// There is no transaction: the event is published after the provider accepted
// the message, at-most-once.
func new[[.Entity | pascal]]Service(cfg *config.Config, infra Infrastructure) application.[[.Entity | pascal]]Service {
	gw := gateway.NewHTTP[[.Entity | pascal]]Gateway(cfg.Gateway.URL, cfg.Gateway.Token)
	publisher := events.NewBusPublisher(infra.EventBus)

	return application.New[[.Entity | pascal]]AppService(gw, publisher)
}
[[- else if eq .Shape "read-model"]]

// new[[.Entity | pascal]]Service builds the repository over the unit of work's
// executor; a read model dispatches nothing.
func new[[.Entity | pascal]]Service(_ *config.Config, infra Infrastructure) application.[[.Entity | pascal]]Service {
	repo := postgres.NewPostgres[[.Entity | pascal]]Repository(pfuow.New(infra.DBPool))

	return application.New[[.Entity | pascal]]AppService(repo)
}
[[- else]]

// new[[.Entity | pascal]]Service builds the repository, the outbox dispatcher and the
// unit of work. The repository and the dispatcher receive the same unit of
// work so a command's row change and its events share one transaction.
func new[[.Entity | pascal]]Service(_ *config.Config, infra Infrastructure) application.[[.Entity | pascal]]Service {
	uow := pfuow.New(infra.DBPool)
	repo := postgres.NewPostgres[[.Entity | pascal]]Repository(uow)
	dispatcher := events.NewPostgresOutboxDispatcher(uow)

	return application.New[[.Entity | pascal]]AppService(repo, uow, dispatcher)
}
[[- end]]

// [[.Entity | pascal]]Service returns the [[.Entity]] slice's in-process API, the
// contract other modules depend on.
func (m *Module) [[.Entity | pascal]]Service() [[.Name]]def.[[.Entity | pascal]]Service {
	return inproc.New[[.Entity | pascal]]ContractAdapter(m.[[.Entity]])
}
