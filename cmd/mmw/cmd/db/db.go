package db

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "db",
		Short:         "Database tools",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(NewDumpCmd())

	return cmd
}
