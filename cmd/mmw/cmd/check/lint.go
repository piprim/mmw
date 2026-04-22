package check

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/piprim/mmw/pkg/platform"
	"github.com/spf13/cobra"
)

func NewLintCmd() *cobra.Command {
	var workspace bool

	cmd := &cobra.Command{
		Use:   "lint [packages...]",
		Short: "Run golangci-lint against Go packages",
		Long: `Runs golangci-lint run against the specified packages.
Linting runs at package level (not per-file) so all linters fire correctly,
including package-scope linters such as revive argument-limit.
golangci-lint must be installed and on PATH.

Defaults to ./... when no package arguments are given.
Use --workspace to lint all modules declared in go.work.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if workspace {
				return lintWorkspace(cmd)
			}

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

	cmd.Flags().BoolVar(&workspace, "workspace", false, "Lint all modules declared in go.work")

	return cmd
}

func lintWorkspace(cmd *cobra.Command) error {
	root := platform.RootRepo()

	modules, err := goWorkModules(root)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	var failed []string

	for _, mod := range modules {
		fmt.Fprintf(out, "\n── lint %s ──\n", mod)

		checker := checks.NewLintCheckerAt(filepath.Join(root, mod), out, errOut)

		result, err := checker.Check(cmd.Context(), nil)
		if err != nil {
			return err
		}

		if result.HasViolations() {
			failed = append(failed, mod)
		}
	}

	fmt.Fprintln(out)

	if len(failed) > 0 {
		return fmt.Errorf("lint violations in: %s", strings.Join(failed, ", "))
	}

	return nil
}

// goWorkModules parses the use directives from go.work and returns the
// relative module paths in declaration order.
func goWorkModules(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		return nil, fmt.Errorf("read go.work: %w", err)
	}

	var modules []string
	inUse := false

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "use (":
			inUse = true
		case inUse && line == ")":
			inUse = false
		case inUse && line != "":
			modules = append(modules, filepath.Clean(line))
		case strings.HasPrefix(line, "use ") && !strings.HasSuffix(line, "("):
			modules = append(modules, filepath.Clean(strings.TrimSpace(strings.TrimPrefix(line, "use "))))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse go.work: %w", err)
	}

	if len(modules) == 0 {
		return nil, errors.New("no modules found in go.work")
	}

	return modules, nil
}
