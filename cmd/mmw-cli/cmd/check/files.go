package check

import (
	"errors"
	"fmt"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewFilesCmd() *cobra.Command {
	var fix bool

	cmd := &cobra.Command{
		Use:   "files [files...]",
		Short: "Check files for trailing whitespace, missing EOF newline, and size > 500 KB",
		Long: `Check each file for:
  - trailing whitespace on any line
  - missing newline at end of file
  - file size > 500 KB (always reported; --fix has no effect)

Defaults to all git-tracked files when no file arguments are given.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checker := checks.NewFilesChecker()

			if fix {
				fixer, ok := checker.(checks.Fixer)
				if !ok {
					return errors.New("files: --fix not supported by this checker")
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

	cmd.Flags().BoolVar(&fix, "fix", false, "rewrite files in-place (strips trailing whitespace, adds EOF newline)")

	return cmd
}

// printResult writes violations to stdout and returns an error when violations exist.
func printResult(cmd *cobra.Command, result checks.Result) error {
	for _, v := range result.Violations {
		loc := v.File
		if v.Line > 0 {
			loc = fmt.Sprintf("%s:%d", v.File, v.Line)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", loc, v.Message)
	}

	if result.HasViolations() {
		return fmt.Errorf("%s: %d violation(s) found", result.CheckerName, len(result.Violations))
	}

	return nil
}
