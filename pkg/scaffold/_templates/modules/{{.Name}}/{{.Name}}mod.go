// Package {{.Name}} is the {{.Name | pascal}} module.
package {{.Name}}

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	pfcore "{{.PlatformPath}}/pkg/platform/core"
	pfoutbox "{{.PlatformPath}}/pkg/platform/db/outbox"
	pfevents "{{.PlatformPath}}/pkg/platform/events"
	pfuow "{{.PlatformPath}}/pkg/platform/pg/uow"
	pfserver "{{.PlatformPath}}/pkg/platform/server"
	"{{.OrgPrefix}}/{{.Name}}/internal/adapters/outbound/events"
	"{{.OrgPrefix}}/{{.Name}}/internal/adapters/outbound/persistence/postgres"
	"{{.OrgPrefix}}/{{.Name}}/internal/application"
	"{{.OrgPrefix}}/{{.Name}}/internal/infra/config"
	"golang.org/x/sync/errgroup"
	"github.com/rotisserie/eris"
)

const (
	relayTableName = "{{.Name}}.event"
	ModuleName     = "{{.Name | pascal}}"
	PGSchema       = "{{.Name}}"
)

var _ pfcore.Module = (*Module)(nil)

type Module struct {
	relay   *pfoutbox.EventsRelay
	server  *pfserver.HTTPServer
	logger  *slog.Logger
	service application.{{.Name | pascal}}Service
}

type Infrastructure struct {
	DBPool   *pgxpool.Pool
	EventBus pfevents.SystemEventBus
	Logger   *slog.Logger
}

func (m *Module) Service() application.{{.Name | pascal}}Service { return m.service }

func New(infra Infrastructure) (*Module, error) {
	cfg, err := config.Load(context.Background(), "")
	if err != nil {
		return nil, eris.Wrap(err, "load {{.Name}} config")
	}

	uow := pfuow.New(infra.DBPool)
	repo := postgres.New{{.Name | pascal}}Repository(uow)
	dispatcher := events.NewPostgresOutboxDispatcher(uow)
	svc := application.New{{.Name | pascal}}ApplicationService(repo, uow, dispatcher)

	relay := pfoutbox.NewEventsRelay(infra.DBPool, infra.EventBus, infra.Logger, relayTableName)
	server := pfserver.NewHTTPServer(pfserver.HTTPServerInfra{
		Config:  cfg.Server,
		Logger:  infra.Logger,
	})

	return &Module{relay: relay, server: server, logger: infra.Logger, service: svc}, nil
}

func (m *Module) Start(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error { return m.server.Start(gCtx) })
	g.Go(func() error { m.relay.Start(gCtx); return nil })
	return g.Wait()
}

func (m *Module) Close() error { return nil }
