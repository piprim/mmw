package check

import (
	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewYAMLCmd() *cobra.Command {
	return newCheckerCmd(
		"yaml [files...]",
		"Check YAML files for syntax errors using yamllint",
		`Runs yamllint -d relaxed against each .yaml/.yml file.
yamllint must be installed and on PATH.

Defaults to all tracked *.yaml/*.yml files when no arguments are given.`,
		checks.NewYAMLChecker,
		"",
	)
}
