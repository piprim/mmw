package workspace

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/piprim/mmw/pkg/platform"
	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "status",
		Short:         "Verify all workspace modules and sync the workspace",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(cmd.Context(), cmd)
		},
	}

	return cmd
}

func runStatus(ctx context.Context, cmd *cobra.Command) error {
	root := platform.RootRepo()
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	modules, err := workModules(root)
	if err != nil {
		return fmt.Errorf("listing workspace modules: %w", err)
	}

	var failed []string

	for _, mod := range modules {
		modDir := filepath.Join(root, mod)
		verify := exec.CommandContext(ctx, "go", "mod", "verify")
		verify.Dir = modDir
		verify.Stdout = out
		verify.Stderr = errOut

		if verifyErr := verify.Run(); verifyErr != nil {
			fmt.Fprintf(out, "  ✗ %s\n", mod)
			failed = append(failed, mod)

			continue
		}

		fmt.Fprintf(out, "  ✓ %s\n", mod)
	}

	if err := runGoCmd(ctx, ioStreams{out, errOut}, root, "work", "sync"); err != nil {
		return fmt.Errorf("go work sync: %w", err)
	}

	if len(failed) > 0 {
		return fmt.Errorf("verification failed: %s", strings.Join(failed, ", "))
	}

	return nil
}
