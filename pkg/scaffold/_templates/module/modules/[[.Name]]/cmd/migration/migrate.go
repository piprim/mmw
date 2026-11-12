package main

import (
	"fmt"
	"log/slog"
	"os"

	pfconfig "[[.PlatformPath]]/pkg/platform/config"
	dbpgcli "[[.PlatformPath]]/pkg/platform/db/cli"
	pfslog "[[.PlatformPath]]/pkg/platform/slog"

	"github.com/rotisserie/eris"

	"[[.ModulePath]]/internal/infra/config"
	"[[.ModulePath]]/internal/infra/persistence/migrations"
)

// main must not import the module root: the root imports the postgres
// adapter, which does not build until boa:generate has run on the table this
// command creates.
func main() {
	conf, err := loadConfig()
	if err != nil {
		logError("loading config failed", err)
		os.Exit(1)
	}

	err = pfconfig.RequireSecret(conf.Environment, "database password", conf.Database.Password, config.SecretPlaceholders...)
	if err != nil {
		logError("loading config failed", eris.Wrap(err, "set it in the TOML overlay or DB_PASSWORD"))
		os.Exit(1)
	}

	if err := dbpgcli.Migrate(conf.Database.URL(), migrations.Schema, migrations.FS); err != nil {
		logError("command failed", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func logError(msg string, err error) {
	logger := slog.New(pfslog.StderrTxtHandler(slog.LevelError, nil))
	logger.Error(msg)
	// Print the formatted stack trace directly to stderr
	_, _ = fmt.Fprintf(os.Stderr, "%s\n", eris.ToString(err, true))
}
