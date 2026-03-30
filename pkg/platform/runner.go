package platform

import (
	"context"
	"log/slog"

	"github.com/piprim/mmw/platform/core"
	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"
)

type App struct {
	logger  *slog.Logger
	modules []core.Module
}

func New(logger *slog.Logger, modules []core.Module) *App {
	return &App{logger: logger, modules: modules}
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting platform runner")

	g, gCtx := errgroup.WithContext(ctx)

	for _, m := range a.modules {
		g.Go(func() error {
			return eris.Wrapf(m.Start(gCtx), "module failed")
		})
	}

	if err := g.Wait(); err != nil {
		msg := "application stopped with error"
		a.logger.Error(msg, "err", err)

		return eris.Wrap(err, msg)
	}

	a.logger.Info("application stopped gracefully")

	return nil
}
