package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
[[- if .Database]]
	"github.com/jackc/pgx/v5/pgxpool"
[[- end]]
	"[[.PlatformPath]]/pkg/platform"
	pfcore "[[.PlatformPath]]/pkg/platform/core"
	pfevents "[[.PlatformPath]]/pkg/platform/events"
[[- if .Database]]
	pfpg "[[.PlatformPath]]/pkg/platform/pg"
[[- end]]
	pfslog "[[.PlatformPath]]/pkg/platform/slog"
	"github.com/rotisserie/eris"
	defauth "[[.ContractsPath]]/go/application/auth"
	"[[.ContractsPath]]/go/network/auth/v1/authv1connect"

	"[[.OrgPrefix]]/[[.Name]]"
	"[[.OrgPrefix]]/[[.Name]]/internal/infra/config"
)

const (
	outputChannelBufferSize = 1024
)

var errFormater = eris.ToJSON

var (
	logger   *slog.Logger
[[- if .Database]]
	dbPool   *pgxpool.Pool
[[- end]]
	exitCode = 0
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		cancel()
[[- if .Database]]
		if dbPool != nil {
			dbPool.Close()
		}
[[- end]]

		os.Exit(exitCode)
	}()

	var err error

	conf, err := config.Load(ctx, "")
	if err != nil {
		exitCode = 1
		_, _ = fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	logger, err = pfslog.New(pfslog.HandlerText, conf.LogLevel.SlogLevel())
	if err != nil {
		exitCode = 1
		_, _ = fmt.Fprint(os.Stdout, eris.ToString(err, true)+"\n")

		return
	}

	moduleLogger := logger.With("module", [[.Name]].ModuleName)

	watermillLogger := watermill.NewSlogLogger(moduleLogger)
	rawBus := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: outputChannelBufferSize,
			// Persistent guarantees the channel won't drop messages if no subscriber is attached yet
			Persistent: true,
		},
		watermillLogger,
	)

	defer rawBus.Close()
	// Wrap the raw infrastructure in the Adapter.
	systemBus := pfevents.NewWatermillBus(rawBus)

	// When extracted, you might swap Watermill's GoChannel for RabbitMQ here!
	// systemBus := setupRabbitMQ()

[[- if .Database]]

	dbPool, err = pfpg.GetPgxPool(ctx, logger, conf.Database.URL())
	if err != nil {
		logError("creating database pool", err)

		return
	}
[[- end]]

	// authGrpc := authv1connect.NewAuthPrivateServiceClient(httpClient connect.HTTPClient)
	authHTTPClient := authv1connect.NewAuthPrivateServiceClient(
		&http.Client{}, // no TLS needed for localhost
		conf.AuthServer.URL("", nil),
	)

	module, err := [[.Name]].New(ctx, [[.Name]].Infrastructure{
[[- if .Database]]
		DBPool:     dbPool,
[[- end]]
		EventBus:   systemBus,
		Subscriber: rawBus,
		Logger:     moduleLogger,
		AuthSvc:    defauth.NewAuthPrivateServiceHTTPClient(authHTTPClient),
	})
	if err != nil {
		logError("creating module failed", err)

		return
	}

	modules := []pfcore.Module{module}

	err = platform.New(logger, modules).Run(ctx)
	if err != nil {
		logError("platform error", err)
		exitCode = 1
	}
}

func logError(msg string, err error) {
	l := slog.New(pfslog.StderrTxtHandler(slog.LevelDebug, nil))
	l.Error(msg, "details", errFormater(err, true))
}
