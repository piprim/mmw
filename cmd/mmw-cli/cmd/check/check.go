package check

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run architectural validators",
	}
	cmd.AddCommand(NewArchCmd())

	return cmd
}
