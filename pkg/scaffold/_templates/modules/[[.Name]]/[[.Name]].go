// Package [[.Name]] is the composition root of the [[.Name | pascal]] module: it
// builds the adapters, injects them into the application service and exposes
// the module to the platform runner.
package [[.Name]]

import (
	"context"
	"log/slog"
[[- if .Grpc]]
	"net/http"
[[- end]]

	[[if .Grpc]]"connectrpc.com/connect"
	[[end]]"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
[[- if .Database]]
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
[[- end]]
	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"

	[[if .Grpc]]pfconnect "[[.PlatformPath]]/pkg/platform/connect"
	[[end]]pfcore "[[.PlatformPath]]/pkg/platform/core"
[[- if .Database]]
	pfdbmigrator "[[.PlatformPath]]/pkg/platform/db/migrator"
	pfoutbox "[[.PlatformPath]]/pkg/platform/db/outbox"
[[- end]]
	pfevents "[[.PlatformPath]]/pkg/platform/events"
[[- if .Grpc]]
	pfmiddleware "[[.PlatformPath]]/pkg/platform/middleware"
[[- end]]
[[- if .Database]]
	pfuow "[[.PlatformPath]]/pkg/platform/pg/uow"
[[- end]]
[[- if .Grpc]]
	pfserver "[[.PlatformPath]]/pkg/platform/server"
	defauth "[[.ContractsPath]]/go/application/auth"
[[- end]]

	[[.Name]]def "[[.ContractsPath]]/go/application/[[.Name]]"
[[- if .Grpc]]
	"[[.ContractsPath]]/go/network/[[.Name]]/v1/[[.Name]]v1connect"
[[- end]]

	[[if .Grpc]]connecthandler "[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/connect"
	[[end]]"[[.OrgPrefix]]/[[.Name]]/internal/adapters/inbound/inproc"
	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/outbound/events"
[[- if .Database]]
	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/outbound/persistence/postgres"
[[- else]]
	"[[.OrgPrefix]]/[[.Name]]/internal/adapters/outbound/gateway"
[[- end]]
	"[[.OrgPrefix]]/[[.Name]]/internal/application"
	"[[.OrgPrefix]]/[[.Name]]/internal/infra/config"
[[- if .Database]]
	"[[.OrgPrefix]]/[[.Name]]/internal/infra/persistence/migrations"
[[- end]]
)

[[- if .Database]]

const (
	relayTableName = "[[.Name]].event"
	ModuleName     = "[[.Name | pascal]]"
	PGSchema       = "[[.Name]]"
)
[[- else]]

const ModuleName = "[[.Name | pascal]]"
[[- end]]

type Module struct {
[[- if .Database]]
	relay   *pfoutbox.EventsRelay
[[- end]]
[[- if .Grpc]]
	server  *pfserver.HTTPServer
[[- end]]
	router  *message.Router
	logger  *slog.Logger
	service application.[[.Entity | pascal]]Service
}

// Ensure Module implements pfcore.Module.
var _ pfcore.Module = (*Module)(nil)

// Infrastructure is what the module needs from the platform. Every field is a
// contract, not an implementation.
type Infrastructure struct {
[[- if .Database]]
	DBPool     *pgxpool.Pool
[[- end]]
	EventBus   pfevents.SystemEventBus
	Subscriber message.Subscriber
[[- if .Grpc]]
	AuthSvc    defauth.AuthPrivateService
[[- end]]
	Logger     *slog.Logger
}
[[- if .Grpc]]

// Handler returns the module's HTTP handler so tests can wrap it in
// httptest.NewServer without starting a real server on a port.
func (m *Module) Handler() http.Handler {
	return m.server.Handler()
}
[[- end]]

// Service returns the module's in-process API, the contract other modules depend on.
func (m *Module) Service() [[.Name]]def.[[.Entity | pascal]]Service {
	return inproc.New[[.Entity | pascal]]ContractAdapter(m.service)
}
[[- if .Database]]

// Migrate runs all pending database migrations of the module.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	m, err := pfdbmigrator.New(db, migrations.FS, "scripts", PGSchema)
	if err != nil {
		return eris.Wrap(err, "failed to create migrator")
	}

	_, err = m.Up(ctx)

	return eris.Wrap(err, "failed to migrate up")
}
[[- end]]

// New wires the dependencies of the module and returns a ready-to-start Module.
func New(ctx context.Context, infra Infrastructure) (*Module, error) {
	cfg, err := config.Load(ctx, "")
	if err != nil {
		return nil, eris.Wrap(err, "app failed to load config")
	}

	service := new[[.Entity | pascal]]Service(cfg, infra)

	router, err := newEventRouter(infra)
	if err != nil {
		return nil, err
	}

[[- if .Database]]

	eventsRelay, err := pfoutbox.NewEventsRelay(infra.DBPool, infra.EventBus, infra.Logger, relayTableName)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create events relay")
	}
[[- end]]

	return &Module{
[[- if .Database]]
		relay:   eventsRelay,
[[- end]]
[[- if .Grpc]]
		server:  newHTTPServer(cfg, infra, service),
[[- end]]
		router:  router,
		logger:  infra.Logger,
		service: service,
	}, nil
}

[[- if .Database]]

// new[[.Entity | pascal]]Service builds the repository, the outbox dispatcher and the
// unit of work. The repository and the dispatcher receive the same unit of
// work so a command's row change and its events share one transaction.
func new[[.Entity | pascal]]Service(_ *config.Config, infra Infrastructure) application.[[.Entity | pascal]]Service {
	uow := pfuow.New(infra.DBPool)
	repo := postgres.NewPostgres[[.Entity | pascal]]Repository(uow)
	dispatcher := events.NewPostgresOutboxDispatcher(uow)

	return application.New[[.Entity | pascal]]AppService(repo, uow, dispatcher)
}
[[- else]]

// new[[.Entity | pascal]]Service builds the HTTP gateway and the bus publisher.
// There is no transaction: the event is published after the provider accepted
// the message, at-most-once.
func new[[.Entity | pascal]]Service(cfg *config.Config, infra Infrastructure) application.[[.Entity | pascal]]Service {
	gw := gateway.NewHTTP[[.Entity | pascal]]Gateway(cfg.Gateway.URL, cfg.Gateway.Token)
	publisher := events.NewBusPublisher(infra.EventBus)

	return application.New[[.Entity | pascal]]AppService(gw, publisher)
}
[[- end]]

// newEventRouter creates the Watermill router. No inbound handler is
// registered yet: add them here (adapters/inbound/events) when the module
// subscribes to another module's events.
func newEventRouter(infra Infrastructure) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermill.NewSlogLogger(infra.Logger))
	if err != nil {
		return nil, eris.Wrap(err, "failed to create event router")
	}

	return router, nil
}

[[- if .Grpc]]

// newHTTPServer mounts the Connect handler behind the bearer-auth middleware
// (every RPC needs a valid JWT), under its own path and under /api.
func newHTTPServer(
	cfg *config.Config, infra Infrastructure, service application.[[.Entity | pascal]]Service,
) *pfserver.HTTPServer {
	mux := http.NewServeMux()

	interceptors := connect.WithInterceptors(pfconnect.NewErrorLoggingInterceptor(infra.Logger))
	path, handler := [[.Name]]v1connect.New[[.Entity | pascal]]ServiceHandler(
		connecthandler.New[[.Entity | pascal]]Handler(service), interceptors,
	)

	authMiddleware := pfmiddleware.BearerAuthMiddleware(
		connecthandler.NewTokenValidator(infra.AuthSvc),
		infra.Logger,
		nil, // no excluded paths: all routes are protected
	)
	mux.Handle(path, authMiddleware(handler))
	mux.Handle("/api"+path, http.StripPrefix("/api", authMiddleware(handler)))

	return pfserver.NewHTTPServer(pfserver.HTTPServerInfra{
		Config:       cfg.Server,
		Handler:      mux,
		Logger:       infra.Logger,
		HealthFns:    pfserver.HealthFns{"[[if .Database]]database[[else]]gateway[[end]]": service.Health},
		LogPayloads:  true,
		ServiceNames: []string{[[.Name]]v1connect.[[.Entity | pascal]]ServiceName}, // gRPC reflection
	})
}
[[- end]]

// Close releases the module's own resources (none yet).
func (m *Module) Close() error {
	m.logger.Info("shutting down module internal resources")

	return nil
}

// Start runs the module's goroutines until ctx is cancelled or one of them fails.
func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("starting the app")

	g, gCtx := errgroup.WithContext(ctx)

[[- if .Grpc]]

	g.Go(func() error {
		return m.server.Start(gCtx)
	})
[[- end]]
[[- if .Database]]

	if m.relay != nil {
		g.Go(func() error {
			m.relay.Start(gCtx)

			return nil
		})
	}
[[- end]]

	g.Go(func() error {
		return m.router.Run(gCtx)
	})

	return eris.Wrapf(g.Wait(), "%s failure", ModuleName)
}
