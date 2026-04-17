package check

import (
	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewTOMLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "toml [files...]",
		Short: "Check TOML files for syntax errors",
		Long: `Parses each .toml file using go-toml/v2 and reports syntax errors.
No subprocess needed — go-toml/v2 is used as a library.

Defaults to all tracked *.toml files when no arguments are given.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checker := checks.NewTOMLChecker()

			result, err := checker.Check(cmd.Context(), args)
			if err != nil {
				return err
			}

			return printResult(cmd, result)
		},
	}
}
