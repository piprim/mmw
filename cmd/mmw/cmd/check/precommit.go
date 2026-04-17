package check

import (
	"errors"
	"fmt"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewPreCommitCmd() *cobra.Command {
	var modified bool
	var failFast bool

	cmd := &cobra.Command{
		Use:   "pre-commit",
		Short: "Run all pre-commit checks (read-only orchestrator)",
		Long: `Orchestrates all check commands against git-selected files.

File selection:
  default    staged files only   (git diff --cached --name-only --diff-filter=ACM)
  --modified staged + modified tracked files (suitable for manual runs)

Check order:
  1. files  — trailing whitespace, EOF newline, size > 500 KB
  2. yaml   — YAML syntax (yamllint)
  3. toml   — TOML syntax (go-toml/v2)
  4. format — gofumpt formatting
  5. lint   — golangci-lint

All checks run even when an earlier one fails (use --fail-fast to stop early).
This command is read-only: it never modifies files or alters the git index.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			files, err := checks.SelectFiles(ctx, modified)
			if err != nil {
				return err
			}

			if len(files) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no staged or modified files to check")

				return nil
			}

			// Build the ordered checker list.  Packages for the lint checker are
			// derived from the .go files in the selected file set.
			goPackages := checks.PackageDirsFromFiles(files)

			allCheckers := []checks.Checker{
				checks.NewFilesChecker(),
				checks.NewYAMLChecker(),
				checks.NewTOMLChecker(),
				checks.NewFormatChecker(),
			}

			var hasViolations bool

			results, err := checks.RunPreCommit(ctx, allCheckers, files, failFast)
			if err != nil {
				return err
			}

			for _, result := range results {
				if result.HasViolations() {
					hasViolations = true
				}

				for _, v := range result.Violations {
					loc := v.File
					if v.Line > 0 {
						loc = fmt.Sprintf("%s:%d", v.File, v.Line)
					}

					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", result.CheckerName, loc, v.Message)
				}
			}

			// Run lint separately if we have Go files and haven't fail-fasted.
			// golangci-lint output is streamed directly to stdout — no buffering or parsing.
			if len(goPackages) > 0 && !(failFast && hasViolations) {
				lintResult, err := checks.NewLintChecker(cmd.OutOrStdout(), cmd.ErrOrStderr()).Check(ctx, goPackages)
				if err != nil {
					return err
				}

				if lintResult.HasViolations() {
					hasViolations = true
				}
			}

			if hasViolations {
				return errors.New("pre-commit: violations found")
			}

			fmt.Fprintln(cmd.OutOrStdout(), "pre-commit: all checks passed")

			return nil
		},
	}

	cmd.Flags().BoolVar(&modified, "modified", false, "include modified tracked files (staged + unstaged)")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "stop after the first checker that reports violations")

	return cmd
}
