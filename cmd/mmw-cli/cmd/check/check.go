package check

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "check",
		Short:         "Run validators and pre-commit checks",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(NewArchCmd())
	cmd.AddCommand(NewFilesCmd())
	cmd.AddCommand(NewFormatCmd())
	cmd.AddCommand(NewTOMLCmd())
	cmd.AddCommand(NewYAMLCmd())
	cmd.AddCommand(NewLintCmd())
	cmd.AddCommand(NewPreCommitCmd())

	return cmd
}
