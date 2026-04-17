package test

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "test",
		Short: "Run test helpers",
	}

	rootCmd.AddCommand(NewCoverageCmd())

	return rootCmd
}
