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
			exitCode := archtest.RunAll(root)
			err := error(nil)

			if exitCode != 0 {
				err = errors.New("arch test failed")
			}

			return err
		},
	}
}
