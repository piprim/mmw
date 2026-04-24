package check

import (
	"errors"
	"fmt"
	"strings"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

var errFixNotSupported = errors.New("--fix not supported by this checker")

func NewFilesCmd() *cobra.Command {
	return newCheckerCmd(
		"files [files...]",
		"Check files for trailing whitespace, missing EOF newline, and size > 500 KB",
		`Check each file for:
  - trailing whitespace on any line
  - missing newline at end of file
  - file size > 500 KB (always reported; --fix has no effect)

Defaults to all git-tracked files when no file arguments are given.`,
		checks.NewFilesChecker,
		"rewrite files in-place (strips trailing whitespace, adds EOF newline)",
	)
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

// newCheckerCmd builds a cobra.Command that runs newChecker and optionally
// fixes in-place when --fix is set. use must start with the command name
// (used for error messages), e.g. "files [files...]".
func newCheckerCmd(
	use, short, long string,
	newChecker func() checks.Checker,
	fixFlagDesc string,
) *cobra.Command {
	name := strings.Fields(use)[0]
	var fix bool

	cmd := &cobra.Command{
		Use:           use,
		Short:         short,
		Long:          long,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checker := newChecker()

			if fix {
				fixer, ok := checker.(checks.Fixer)
				if !ok {
					return fmt.Errorf("%s: %w", name, errFixNotSupported)
				}

				if err := fixer.Fix(cmd.Context(), args); err != nil {
					return fmt.Errorf("%s fix: %w", name, err)
				}

				return nil
			}

			result, err := checker.Check(cmd.Context(), args)
			if err != nil {
				return fmt.Errorf("%s check: %w", name, err)
			}

			return printResult(cmd, result)
		},
	}

	if fixFlagDesc != "" {
		cmd.Flags().BoolVar(&fix, "fix", false, fixFlagDesc)
	}

	return cmd
}
