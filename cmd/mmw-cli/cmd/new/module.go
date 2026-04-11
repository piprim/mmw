package new

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
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
	cmd.Flags().StringVar(&templatePath, "template", "", "Path to an external template directory (default: embedded templates)")

	return cmd
}

func runNewModule(templatePath string) error {
	root := platform.RootRepo()

	fsys, err := selectTemplateFS(templatePath)
	if err != nil {
		return err
	}

	m, err := scaffold.LoadManifest(fsys)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	// Seed OrgPrefix default from contracts/go.mod detection.
	for i, v := range m.Variables {
		if v.Name == "OrgPrefix" {
			detected := detectOrgPrefix(root)
			if detected != "" {
				m.Variables[i].Default = detected
			}
		}
	}

	vars, err := collectVars(m)
	if err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	if err := scaffold.EnrichVars(vars); err != nil {
		return err
	}

	name, _ := vars["Name"].(string)

	if err := scaffold.GenerateModule(fsys, root, vars); err != nil {
		return fmt.Errorf("generate module: %w", err)
	}

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

// collectVars builds and runs a huh form from the manifest variables,
// returning a PascalCase-keyed map of collected values.
func collectVars(m *scaffold.Manifest) (map[string]any, error) {
	vars := make(map[string]any, len(m.Variables))

	// Pre-populate with defaults so all keys exist before the form runs.
	for _, v := range m.Variables {
		vars[v.Name] = v.Default
	}

	// Use pointer-backed approach: each field writes into a dedicated pointer.
	// After the form runs, the pointers contain the final values.
	type binding struct {
		name  string
		apply func()
	}
	var bindings []binding
	var fields []huh.Field

	for i := range m.Variables {
		v := m.Variables[i]
		switch v.Kind {
		case scaffold.KindText:
			val := ""
			if s, ok := v.Default.(string); ok {
				val = s
			}
			ptr := &val
			fields = append(fields, huh.NewInput().
				Title(v.Name).
				Value(ptr).
				Validate(func(s string) error {
					if v.Default.(string) == "" && s == "" {
						return fmt.Errorf("%s is required", v.Name)
					}
					return nil
				}),
			)
			name := v.Name
			bindings = append(bindings, binding{name: name, apply: func() { vars[name] = *ptr }})

		case scaffold.KindBool:
			val := false
			if b, ok := v.Default.(bool); ok {
				val = b
			}
			ptr := &val
			fields = append(fields, huh.NewConfirm().
				Title(v.Name).
				Value(ptr),
			)
			name := v.Name
			bindings = append(bindings, binding{name: name, apply: func() { vars[name] = *ptr }})

		case scaffold.KindChoice:
			choices, _ := v.Default.([]string)
			opts := make([]huh.Option[string], len(choices))
			for j, c := range choices {
				opts[j] = huh.NewOption(c, c)
			}
			val := ""
			if len(choices) > 0 {
				val = choices[0]
			}
			ptr := &val
			fields = append(fields, huh.NewSelect[string]().
				Title(v.Name).
				Options(opts...).
				Value(ptr),
			)
			name := v.Name
			bindings = append(bindings, binding{name: name, apply: func() { vars[name] = *ptr }})
		}
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	if err := form.Run(); err != nil {
		return nil, err
	}

	// Apply bindings: write pointer values into the vars map.
	for _, b := range bindings {
		b.apply()
	}

	return vars, nil
}
