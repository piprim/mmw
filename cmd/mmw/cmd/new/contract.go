package new

import (
	"fmt"

	"github.com/piprim/goplt"
	"github.com/piprim/mmw/pkg/platform"
	"github.com/spf13/cobra"
)

func NewContractCmd() *cobra.Command {
	var templatePath string

	cmd := &cobra.Command{
		Use:   "contract <name>",
		Short: "Generate a contract definition for an existing module",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runNewContract(args[0], templatePath)
		},
	}
	cmd.Flags().StringVar(&templatePath, "template", "", "Path to an external template directory")

	return cmd
}

func runNewContract(name, templatePath string) error {
	root := platform.RootRepo()

	fsys, err := selectTemplateFS(templatePath)
	if err != nil {
		return err
	}

	m, err := goplt.LoadManifest(fsys)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	vars := map[string]any{
		"Name":      name,
		"OrgPrefix": detectOrgPrefix(root),
		// ContractsPath and PlatformPath are only referenced by templates under
		// modules/ — which is skipped when WithModule=false. Safe to omit.
		"ContractsPath": "",
		"PlatformPath":  "",
		"WithModule":    false, // skip modules/ subtree
		"WithConnect":   true,
		"WithContract":  true,
		"WithDatabase":  false,
		"License":       "MIT",
	}

	if err := goplt.NewGenerator().Generate(fsys, m, root, vars); err != nil {
		return fmt.Errorf("generate contract: %w", err)
	}

	fmt.Printf("✓ Contract definition generated in contracts/definitions/%s/\n", name)

	return nil
}
