package new

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/piprim/goplt"
	gopltui "github.com/piprim/goplt/tui"
	"github.com/piprim/mmw/pkg/platform"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/spf13/cobra"
)

var (
	errorC   = color.New(color.FgRed)
	successC = color.New(color.FgGreen)
	infoC    = color.New(color.FgBlue)
	warnC    = color.New(color.FgYellow)
)

func NewModuleCmd() *cobra.Command {
	var templatePath string

	cmd := &cobra.Command{
		Use:   "module",
		Short: "Scaffold a new module interactively",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runNewModule(templatePath)
		},
	}
	cmd.Flags().StringVar(
		&templatePath, "template", "",
		"Path to an external template directory (default: embedded templates)",
	)

	return cmd
}

func runNewModule(templatePath string) error {
	root := platform.RootRepo()

	fsys, err := selectTemplateFS(templatePath)
	if err != nil {
		return err
	}

	m, err := goplt.LoadManifest(fsys)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Seed OrgPrefix default from workspace detection before the TUI runs.
	for i, v := range m.Variables {
		if v.Name != "OrgPrefix" {
			continue
		}

		if d := detectOrgPrefix(root); d != "" {
			m.Variables[i].Default = d
		}
	}

	vars, err := gopltui.CollectVars(m)
	if err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	// WithModule is a routing flag — not user-facing, drives the modules/ condition.
	vars["WithModule"] = true

	gen := goplt.NewGenerator()
	if err := gen.Generate(fsys, m, root, vars); err != nil {
		return fmt.Errorf("generate module: %w", err)
	}

	if err := goplt.RunHooks(m, root); err != nil {
		return fmt.Errorf("post-generate hooks: %w", err)
	}

	name, _ := vars["Name"].(string)

	if err := scaffold.UpdateGoWork(root, name); err != nil {
		_, _ = errorC.Fprintf(os.Stderr, "warning: could not update go.work: %v\n", err)
	}
	if err := scaffold.UpdateMiseToml(root, name); err != nil {
		_, _ = errorC.Fprintf(os.Stderr, "warning: could not update mise.toml: %v\n", err)
	}

	successC.Printf("\n✓ Module %q generated in modules/%s/\n", name, name)
	warnC.Println("Next steps:")
	infoC.Printf("  - cd modules/%s && go mod tidy\n", name)
	infoC.Println("  - cd <repo-root> && mise run stow && go work sync")

	withContract, _ := vars["WithContract"].(bool)
	if withContract {
		infoC.Println("  - cd contracts && buf generate")
	}
	infoC.Println("  - Wire the module in cmd/mmw/main.go")

	return nil
}
