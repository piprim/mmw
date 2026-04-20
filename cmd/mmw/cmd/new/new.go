package new

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold new modules or contracts",
	}
	cmd.AddCommand(NewModuleCmd())
	cmd.AddCommand(NewContractCmd())

	return cmd
}

// selectTemplateFS returns the embedded FS (default) or an OS directory FS.
func selectTemplateFS(templatePath string) (fs.FS, error) {
	if templatePath == "" {
		return scaffold.EmbeddedFS(), nil
	}
	info, err := os.Stat(templatePath)
	if err != nil {
		return nil, fmt.Errorf("template path %q: %w", templatePath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("template path %q is not a directory", templatePath)
	}

	return os.DirFS(templatePath), nil
}

// detectOrgPrefix reads the contracts go.mod to derive the org prefix.
// Falls back to "github.com/acme" if detection fails.
func detectOrgPrefix(repoRoot string) string {
	const fallback = "github.com/acme"
	data, err := os.ReadFile(filepath.Join(repoRoot, "contracts", "go.mod"))
	if err != nil {
		return fallback
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "module ") {
			continue
		}
		mod := strings.TrimPrefix(line, "module ")
		parts := strings.SplitN(strings.TrimSpace(mod), "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}

	return fallback
}
