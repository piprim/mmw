package check

import (
	"errors"

	"github.com/piprim/mmw/pkg/archtest"
	"github.com/piprim/mmw/pkg/platform"
	"github.com/spf13/cobra"
)

func NewArchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "arch",
		Short: "Validate architectural boundaries",
		RunE: func(_ *cobra.Command, _ []string) error {
			root := platform.RootRepo()
			if archtest.RunAll(root) != 0 {
				return errors.New("arch test failed")
			}

			return nil
		},
	}
}
