// Package [[.Name]] is the composition root of the [[.Name | pascal]] module: it
// builds each slice's adapters (<entity>_wiring.go), injects them into the
// slice's application service and exposes the module to the platform runner.
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
[[- if .Grpc]]
	pfserver "[[.PlatformPath]]/pkg/platform/server"
	defauth "[[.ContractsPath]]/go/application/auth"
[[- end]]

[[- if .Grpc]]

	"[[.ContractsPath]]/go/network/[[.Name]]/v1/[[.Name]]v1connect"
[[- end]]

	[[if .Grpc]]connecthandler "[[.ModulePath]]/internal/adapters/inbound/connect"
	[[end]]"[[.ModulePath]]/internal/application"
	"[[.ModulePath]]/internal/infra/config"
[[- if .Database]]
	"[[.ModulePath]]/internal/infra/persistence/migrations"
[[- end]]
)

[[- if .Database]]

const (
	relayTableName = "[[.Name]].event"
	ModuleName     = "[[.Name | pascal]]"
	PGSchema       = migrations.Schema
)
[[- else]]

const ModuleName = "[[.Name | pascal]]"
[[- end]]

type Module struct {
[[- if .Database]]
	relay  *pfoutbox.EventsRelay
[[- end]]
[[- if .Grpc]]
	server *pfserver.HTTPServer
[[- end]]
	router *message.Router
	logger *slog.Logger
	// mmw:slice-field
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

	m := &Module{
[[- if .Database]]
		relay:  eventsRelay,
[[- end]]
		router: router,
		logger: infra.Logger,
	}
	// mmw:slice-wire
[[- if .Grpc]]
	m.server = newHTTPServer(cfg, infra, m)
[[- end]]

	return m, nil
}

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

// newHTTPServer mounts every slice's Connect handler behind the bearer-auth
// middleware (every RPC needs a valid JWT), under its own path and under /api.
func newHTTPServer(cfg *config.Config, infra Infrastructure, m *Module) *pfserver.HTTPServer {
	mux := http.NewServeMux()
	interceptors := connect.WithInterceptors(pfconnect.NewErrorLoggingInterceptor(infra.Logger))
	authMiddleware := pfmiddleware.BearerAuthMiddleware(
		connecthandler.NewTokenValidator(infra.AuthSvc),
		infra.Logger,
		nil, // no excluded paths: all routes are protected
	)
	health := pfserver.HealthFns{}
	names := []string{} //nolint:prealloc // gRPC reflection; one append per slice at the marker
	mount := func(path string, handler http.Handler) {
		mux.Handle(path, authMiddleware(handler))
		mux.Handle("/api"+path, http.StripPrefix("/api", authMiddleware(handler)))
	}
	// mmw:slice-handler

	return pfserver.NewHTTPServer(pfserver.HTTPServerInfra{
		Config:       cfg.Server,
		Handler:      mux,
		Logger:       infra.Logger,
		HealthFns:    health,
		LogPayloads:  true,
		ServiceNames: names,
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
		// Run only returns once the router is closed. Cancelling the context stops the
		// handlers, but with no handler registered nothing ever closes the router, so the
		// process would hang on shutdown: close it explicitly.
		stop := context.AfterFunc(gCtx, func() { _ = m.router.Close() })
		defer stop()

		return m.router.Run(gCtx)
	})

	return eris.Wrapf(g.Wait(), "%s failure", ModuleName)
}
