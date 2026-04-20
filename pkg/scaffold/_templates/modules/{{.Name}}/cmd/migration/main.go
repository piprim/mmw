package main

import (
	"fmt"
	"log/slog"
	"os"

	dbpgcli "{{.PlatformPath}}/pkg/platform/db/cli"
	pfslog "{{.PlatformPath}}/pkg/platform/slog"
	{{.Name}} "{{.OrgPrefix}}/{{.Name}}"

	"github.com/rotisserie/eris"
)

func main() {
	if err := dbpgcli.Migrate(nil, {{.Name}}.PGSchema, nil); err != nil {
		logger := slog.New(pfslog.StderrTxtHandler(slog.LevelDebug, nil))
		logger.Error("command failed")
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", eris.ToString(err, true))
		os.Exit(1)
	}
}
