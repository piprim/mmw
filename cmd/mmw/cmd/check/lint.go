package check

import (
	"errors"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewLintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint [packages...]",
		Short: "Run golangci-lint against Go packages",
		Long: `Runs golangci-lint run against the specified packages.
Linting runs at package level (not per-file) so all linters fire correctly,
including package-scope linters such as revive argument-limit.
golangci-lint must be installed and on PATH.

Defaults to ./... when no package arguments are given.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checker := checks.NewLintChecker(cmd.OutOrStdout(), cmd.ErrOrStderr())

			result, err := checker.Check(cmd.Context(), args)
			if err != nil {
				return err
			}

			if result.HasViolations() {
				return errors.New("lint: violations found")
			}

			return nil
		},
	}
}
