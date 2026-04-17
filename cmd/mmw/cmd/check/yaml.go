package check

import (
	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewYAMLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "yaml [files...]",
		Short: "Check YAML files for syntax errors using yamllint",
		Long: `Runs yamllint -d relaxed against each .yaml/.yml file.
yamllint must be installed and on PATH.

Defaults to all tracked *.yaml/*.yml files when no arguments are given.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checker := checks.NewYAMLChecker()

			result, err := checker.Check(cmd.Context(), args)
			if err != nil {
				return err
			}

			return printResult(cmd, result)
		},
	}
}
