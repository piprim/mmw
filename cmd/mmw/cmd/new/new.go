package new

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold a new module",
	}
	cmd.AddCommand(NewModuleCmd())

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
