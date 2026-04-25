package workspace

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/piprim/mmw/pkg/platform"
	"github.com/spf13/cobra"
)

func NewSyncCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "sync [module-dir]",
		Short: "Update a workspace module's commit across all dependents",
		Long: `Syncs a module's HEAD commit hash across all other workspace modules that depend on it.

For a single module:
  1. Runs "git rev-parse HEAD" in the module directory to obtain the commit hash.
  2. Reads the Go module path from the module's go.mod file.
  3. For every other module declared in go.work whose go.mod references that module path,
     runs "go get -u <module>@<commit>" followed by "go mod tidy".
  4. Runs "go work sync" at the workspace root.

With --all, repeats the above for every module declared in go.work, in declaration order.
The positional argument is ignored when --all is set.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := platform.RootRepo()
			ctx := cmd.Context()
			out := cmd.OutOrStdout()
			errOut := cmd.ErrOrStderr()

			if all {
				modules, err := workModules(root)
				if err != nil {
					return err
				}

				for _, mod := range modules {
					if err := syncModule(ctx, out, errOut, root, mod); err != nil {
						return err
					}
				}

				return runCmd(ctx, out, errOut, root, "go", "work", "sync")
			}

			if len(args) == 0 {
				return fmt.Errorf("workspace sync: module-dir argument is required (or use --all)")
			}

			if err := syncModule(ctx, out, errOut, root, args[0]); err != nil {
				return err
			}

			return runCmd(ctx, out, errOut, root, "go", "work", "sync")
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Sync every module in the workspace in declaration order")

	return cmd
}

func syncModule(ctx context.Context, out, errOut io.Writer, root, modPath string) error {
	absModDir := filepath.Join(root, modPath)

	commit, err := gitCommit(absModDir)
	if err != nil {
		return err
	}

	moduleName, err := readModuleName(absModDir)
	if err != nil {
		return err
	}

	shortCommit := commit
	if len(commit) > 8 {
		shortCommit = commit[:8]
	}

	fmt.Fprintf(out, "── sync %s @ %s ──\n", moduleName, shortCommit)

	modules, err := workModules(root)
	if err != nil {
		return err
	}

	moduleRef := fmt.Sprintf("%s@%s", moduleName, commit)

	for _, mod := range modules {
		depDir := filepath.Join(root, mod)
		if depDir == absModDir {
			continue
		}

		goModData, err := os.ReadFile(filepath.Join(depDir, "go.mod"))
		if err != nil {
			return fmt.Errorf("read go.mod in %s: %w", mod, err)
		}

		if !strings.Contains(string(goModData), moduleName) {
			continue
		}

		fmt.Fprintf(out, "  → updating %s\n", mod)

		if err := runCmd(ctx, out, errOut, depDir, "go", "get", moduleRef); err != nil {
			return err
		}

		if err := runCmd(ctx, out, errOut, depDir, "go", "mod", "tidy"); err != nil {
			return err
		}
	}

	return nil
}

func gitCommit(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD in %s: %w", dir, err)
	}

	return strings.TrimSpace(string(out)), nil
}

func readModuleName(dir string) (string, error) {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("open go.mod in %s: %w", dir, err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("parse go.mod in %s: %w", dir, err)
	}

	return "", fmt.Errorf("workspace: no module directive found in %s/go.mod", dir)
}
