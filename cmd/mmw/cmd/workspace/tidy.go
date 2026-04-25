package workspace

import (
	"fmt"
	"path/filepath"

	"github.com/piprim/mmw/pkg/platform"
	"github.com/spf13/cobra"
)

func NewTidyCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "tidy",
		Short:         "Run go mod tidy in every workspace module then go work sync",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()
			errOut := cmd.ErrOrStderr()
			root := platform.RootRepo()

			modules, err := workModules(root)
			if err != nil {
				return err
			}

			for _, mod := range modules {
				fmt.Fprintf(out, "── tidy %s ──\n", mod)

				if err := runCmd(ctx, out, errOut, filepath.Join(root, mod), "go", "mod", "tidy"); err != nil {
					return err
				}
			}

			return runCmd(ctx, out, errOut, root, "go", "work", "sync")
		},
	}
}
