package new

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/piprim/goplt"
	gopltui "github.com/piprim/goplt/tui"
	"github.com/piprim/mmw/pkg/platform"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/spf13/cobra"
)

// moduleVar is the slice manifest variable holding the target module.
const moduleVar = "Module"

func NewEntityCmd() *cobra.Command {
	var (
		module       string
		templatePath string
		varFlags     []string
	)

	cmd := &cobra.Command{
		Use:   "entity [MODULE]",
		Short: "Add a slice (a new entity) to an existing module",
		Long: `Renders one more slice (domain, application, adapters, proto, migrations) into
modules/<module>/, wires it at the module's markers, then runs buf:generate,
go mod tidy and go work sync. Without a module the TUI asks for it.

database and grpc come from the module tree; shape is aggregate, read-model or
gateway (only gateway without a database). Snippets whose marker is missing
(hand-written modules) are printed. An external --template directory holds
module/ and slice/ subdirectories.

  mmw new entity facturation --var entity=avoir --var shape=read-model`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			positional := ""
			if len(args) > 0 {
				positional = args[0]
			}

			return runNewEntity(c.Context(), positional, module, templatePath, varFlags)
		},
	}
	cmd.Flags().StringVar(&module, "module", "", "Module to add the slice to (overrides the argument)")
	cmd.Flags().StringVar(&templatePath, "template", "",
		"Path to an external template directory (default: embedded templates)")
	cmd.Flags().StringArrayVar(&varFlags, "var", nil, "Set a template variable as key=value (repeatable)")

	return cmd
}

func runNewEntity(ctx context.Context, positional, moduleFlag, templatePath string, varFlags []string) error {
	root := platform.RootRepo()

	name, err := resolveModule(root, positional, moduleFlag)
	if err != nil {
		return err
	}
	sliceFS, err := selectTemplateFS(templatePath, "slice")
	if err != nil {
		return err
	}
	m, err := goplt.LoadManifest(sliceFS)
	if err != nil {
		return fmt.Errorf("load slice manifest: %w", err)
	}

	if name == "" {
		if name, err = askModule(root, m); err != nil {
			return err
		}
	}
	// The module is settled; the rest of the prompt does not offer it again.
	m.Variables = slices.DeleteFunc(m.Variables, func(v goplt.Variable) bool { return v.Name == moduleVar })

	traits, err := scaffold.DetectModule(root, name)
	if err != nil {
		return fmt.Errorf("module: %w", err)
	}
	narrowShapes(m, traits.Database)

	vars, err := collectVars(m, varFlags)
	if err != nil {
		return err
	}
	detected, err := scaffold.Detect(root)
	if err != nil {
		return fmt.Errorf("detect workspace: %w", err)
	}
	detected.Apply(vars)
	traits.Apply(vars) // after Detect: ModulePath is the module's own
	scaffold.ApplyShapeDefault(vars)

	if err := scaffold.ValidateSlice(root, vars); err != nil {
		return fmt.Errorf("invalid variables: %w", err)
	}

	report, err := scaffold.AddSlice(sliceFS, root, vars)
	if err != nil {
		return fmt.Errorf("add slice: %w", err)
	}
	if err := scaffold.PostGenerate(ctx, os.Stdout, os.Stderr, root, name); err != nil {
		warnC.Println(err)
		warnC.Println("Next:")
		printSliceNext(name, vars, report)
		printTableNote(name, vars)

		return fmt.Errorf("post-generate: %w", err)
	}

	successC.Printf("\n✓ Slice %q added to modules/%s/\n", vars["Entity"], name)
	warnC.Println("Next:")
	printSliceNext(name, vars, report)
	printTableNote(name, vars)

	return nil
}

// askModule asks only for the module: goplt loads a stringChoice as not
// required, so collectVars would skip the TUI for this one-variable manifest.
func askModule(root string, m *goplt.Manifest) (string, error) {
	modules, err := listModules(root)
	if err != nil {
		return "", err
	}
	if len(modules) == 0 {
		return "", fmt.Errorf("no module under %s/modules; create one with `mmw new module`", root)
	}
	for i := range m.Variables {
		if m.Variables[i].Name != moduleVar {
			continue
		}
		m.Variables[i].Value = modules
		vars, err := gopltui.CollectVars(&goplt.Manifest{Variables: []goplt.Variable{m.Variables[i]}})
		if err != nil {
			return "", fmt.Errorf("prompt cancelled: %w", err)
		}
		name, _ := vars[moduleVar].(string)

		return name, nil
	}

	return "", errors.New("slice manifest has no Module variable")
}

// resolveModule returns the module name: the flag wins over the positional
// argument, and a path is reduced to its base name. Empty means "ask".
//
//nolint:unparam // the error result is part of the planned signature
func resolveModule(_, positional, flag string) (string, error) {
	v := firstNonEmpty(flag, positional)
	v = strings.TrimRight(v, "/")
	if v == "" {
		return "", nil
	}

	return filepath.Base(v), nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}

	return b
}

// listModules lists the directories under modules/ holding <dir>.go, sorted.
func listModules(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "modules"))
	if err != nil {
		return nil, fmt.Errorf("list modules: %w", err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "modules", e.Name(), e.Name()+".go")); err == nil {
			out = append(out, e.Name())
		}
	}
	slices.Sort(out)

	return out, nil
}

// narrowShapes limits the Shape choices to what the module's database allows.
func narrowShapes(m *goplt.Manifest, database bool) {
	for i := range m.Variables {
		if m.Variables[i].Name == "Shape" {
			m.Variables[i].Value = scaffold.AllowedShapes(database)
		}
	}
}
