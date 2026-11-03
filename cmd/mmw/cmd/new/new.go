package new

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Scaffold a new module or slice",
	}
	cmd.AddCommand(NewModuleCmd())
	cmd.AddCommand(NewEntityCmd())

	return cmd
}

// selectTemplateFS returns the embedded FS for sub ("module" or "slice") by
// default, or the sub directory of an external template path.
func selectTemplateFS(templatePath, sub string) (fs.FS, error) {
	if templatePath == "" {
		if sub == "slice" {
			return scaffold.SliceFS(), nil
		}

		return scaffold.ModuleFS(), nil
	}
	dir := filepath.Join(templatePath, sub)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("template path %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("template path %q is not a directory", dir)
	}

	return os.DirFS(dir), nil
}
