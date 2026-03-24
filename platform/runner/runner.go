package oglrunner

import (
	"context"
	"log/slog"

	oglcore "github.com/ovya/ogl/platform/core"
	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"
)

type App struct {
	logger  *slog.Logger
	modules []oglcore.Module
}

func New(logger *slog.Logger, modules []oglcore.Module) *App {
	return &App{logger: logger, modules: modules}
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting platform runner")

	g, gCtx := errgroup.WithContext(ctx)

	// Start all modules concurrently
	for _, m := range a.modules {
		g.Go(func() error {
			// This blocks until the module crashes or gCtx is canceled
			return eris.Wrapf(m.Start(gCtx), "module failed")
		})
	}

	// Wait for all modules to cleanly exit
	if err := g.Wait(); err != nil {
		msg := "application stopped with error"
		a.logger.Error(msg, "err", err)

		return eris.Wrap(err, msg)
	}

	a.logger.Info("application stopped gracefully")

	return nil
}
