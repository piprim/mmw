package new

import (
	"fmt"

	"github.com/piprim/mmw/pkg/platform"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/spf13/cobra"
)

func NewContractCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "contract <name>",
		Short: "Generate a contract definition for an existing module",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			rootRepo := platform.RootRepo()
			opts := scaffold.Options{
				Name:        name,
				OrgPrefix:   detectOrgPrefix(rootRepo),
				RepoRoot:    rootRepo,
				WithConnect: true,
			}
			if err := scaffold.GenerateContract(opts); err != nil {
				return fmt.Errorf("generate contract: %w", err)
			}

			fmt.Printf("✓ Contract definition generated in contracts/definitions/%s/\n", name)

			return nil
		},
	}
}
