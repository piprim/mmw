package check

import (
	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewTOMLCmd() *cobra.Command {
	return newCheckerCmd(
		"toml [files...]",
		"Check TOML files for syntax errors",
		`Parses each .toml file using go-toml/v2 and reports syntax errors.
No subprocess needed — go-toml/v2 is used as a library.

Defaults to all tracked *.toml files when no arguments are given.`,
		checks.NewTOMLChecker,
		"",
	)
}
