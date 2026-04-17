package check

import (
	"errors"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewFormatCmd() *cobra.Command {
	var fix bool

	cmd := &cobra.Command{
		Use:   "format [files...]",
		Short: "Check Go source formatting using gofumpt",
		Long: `Reports any .go file whose content differs from what gofumpt would produce.
No subprocess needed — gofumpt is used as a library.

Defaults to all tracked *.go files when no arguments are given.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checker := checks.NewFormatChecker()

			if fix {
				fixer, ok := checker.(checks.Fixer)
				if !ok {
					return errors.New("format: --fix not supported by this checker")
				}

				return fixer.Fix(cmd.Context(), args)
			}

			result, err := checker.Check(cmd.Context(), args)
			if err != nil {
				return err
			}

			return printResult(cmd, result)
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "rewrite .go files in-place with gofumpt output")

	return cmd
}
