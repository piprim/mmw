package cmd

import (
	"github.com/piprim/mmw/cmd/mmw-cli/cmd/check"
	"github.com/piprim/mmw/cmd/mmw-cli/cmd/new"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mmw",
		Short: "MMW platform CLI",
		Long:  "Developer tools for the MMW modular monolith platform.",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
		},
	}
	root.AddCommand(new.NewCmd())
	root.AddCommand(check.NewCmd())

	return root
}
