package new

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold new modules or contracts",
	}
	cmd.AddCommand(NewModuleCmd())
	cmd.AddCommand(NewContractCmd())

	return cmd
}
