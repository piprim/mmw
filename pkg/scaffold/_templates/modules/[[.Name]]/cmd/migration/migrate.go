package main

import (
	"fmt"
	"log/slog"
	"os"

	dbpgcli "[[.PlatformPath]]/pkg/platform/db/cli"
	pfslog "[[.PlatformPath]]/pkg/platform/slog"
	"[[.OrgPrefix]]/[[.Name]]"

	"github.com/rotisserie/eris"
)

func main() {
	conf, err := loadConfig()
	if err != nil {
		logError("loading config failed", err)
		os.Exit(1)
	}

	if err := dbpgcli.Migrate(conf.Database.URL(), [[.Name]].PGSchema, migrationsFS); err != nil {
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
