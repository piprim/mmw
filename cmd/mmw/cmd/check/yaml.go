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

Only error-level problems are reported. The relaxed profile demotes stylistic
rules such as line-length to warnings, and yamllint exits 0 when only warnings
remain; this check follows that verdict.

Defaults to all tracked *.yaml/*.yml files when no arguments are given.`,
		checks.NewYAMLChecker,
		"",
	)
}
