package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ovya/ogl/oglcore"
	"github.com/ovya/ogl/platform"
	"github.com/rotisserie/eris"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

const (
	readHeaderTimeout = 5 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 30 * time.Second
)

// Pinger is a local interface. The server only needs to know how to ping
// the database for health checks. It cannot execute SQL.
type Pinger interface {
	Ping(ctx context.Context) error
}

type App struct {
	config  platform.Config
	logger  *slog.Logger
	modules []oglcore.Module
	db      Pinger // Interface! No pgxpool leak here.
}

// New creates a new Server Application instance
func New(cfg platform.Config, logger *slog.Logger, db Pinger, modules []oglcore.Module) *App {
	return &App{
		config:  cfg,
		logger:  logger,
		modules: modules,
		db:      db,
	}
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		a.logger.Info("cleaning up module resources")
		for _, m := range a.modules {
			if err := m.Close(); err != nil {
				// We log the error, but we don't return it here because we are in a defer
				a.logger.Error("failed to cleanly close module", "error", err)
			}
		}
	}()

	mux := http.NewServeMux()
	appName := a.config.GetAppName()

	// 1. Platform Routes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := a.db.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			a.logger.Error(prefixMsg(appName, "database connection error"), "error", err)

			return
		}
		w.WriteHeader(http.StatusOK)
		// TODO: How to be sure the database is up?
		fmt.Fprintf(w, `{"app-name":"%s" "status":"healthy","database":"maybe up :)"}`, appName)
	})

	// 2. Module Routes (Delegation)
	for _, m := range a.modules {
		m.RegisterRoutes(mux)
	}

	// 3. Middleware
	var rootHandler http.Handler = mux
	// rootHandler = loggingMiddleware(rootHandler, a.logger)
	// rootHandler = withCORS(a.config, rootHandler)

	port := a.config.GetServerPort()

	server := &http.Server{
		Addr:              port,
		Handler:           h2c.NewHandler(rootHandler, &http2.Server{}),
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}

	g, gCtx := errgroup.WithContext(ctx)

	// Start HTTP Server
	g.Go(func() error {
		a.logger.Info("appName", appName, "starting server", "port", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return eris.Wrap(err, prefixMsg(appName, "server failed"))
		}

		return nil
	})

	// Start Module Workers
	for _, m := range a.modules {
		g.Go(func() error {
			return m.StartWorkers(gCtx)
		})
	}

	// Graceful Shutdown Listener
	g.Go(func() error {
		<-gCtx.Done()
		a.logger.Info("appName", appName, "initiating graceful shutdown")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil {
		msg := "application stopped with error"
		a.logger.Error(msg, "err", err)

		return eris.Wrap(err, msg)
	}

	a.logger.Info("application stopped gracefully")

	return nil
}

func prefixMsg(prefix, msg string) string {
	return prefix + " -- " + msg
}
