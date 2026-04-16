package new

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/piprim/mmw/pkg/platform"
	"github.com/piprim/mmw/pkg/scaffold"
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

	orgPrefix := detectOrgPrefix(root)
	vars := map[string]any{
		"Name":         name,
		"OrgPrefix":    orgPrefix,
		"WithConnect":  true,
		"WithContract": true,
		"WithDatabase": false,
	}
	if err := scaffold.EnrichVars(vars); err != nil {
		return fmt.Errorf("enrich vars: %w", err)
	}

	if err := scaffold.GenerateContract(fsys, root, vars); err != nil {
		return fmt.Errorf("generate contract: %w", err)
	}

	fmt.Printf("✓ Contract definition generated in contracts/definitions/%s/\n", name)

	return nil
}

// detectOrgPrefix reads the contracts go.mod to derive the org prefix.
// Falls back to "github.com/pivaldi" if detection fails.
func detectOrgPrefix(repoRoot string) string {
	const fbRepo = "github.com/pivaldi"
	contractsGoMod := filepath.Join(repoRoot, "contracts", "go.mod")
	data, err := os.ReadFile(contractsGoMod)
	if err != nil {
		return fbRepo
	}
	for _, line := range splitLines(string(data)) {
		if len(line) <= 7 || line[:7] != "module " {
			continue
		}
		mod := line[7:]
		parts := splitPath(mod)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}

	return fbRepo
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

func splitPath(s string) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == '/' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}

	parts = append(parts, s[start:])

	return parts
}
