package new

import (
	"fmt"
	"os"
	"path/filepath"

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
	return &cobra.Command{
		Use:   "module",
		Short: "Scaffold a new module interactively",
		RunE:  runNewModule,
	}
}

// TODO: Ask module go.mod repository
func runNewModule(_ *cobra.Command, _ []string) error {
	var name string
	var withConnect, withContract, withDatabase = true, true, true

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Module Name").
				Description("The name used to generate the module in module/<name>.").Value(&name),
		),
		huh.NewGroup(
			huh.NewConfirm().
				// Inline(true).
				Title("Expose via Connect RPC (HTTP)?").
				Description(" -- Generates the Connect handler + proto file.").
				Value(&withConnect),
			huh.NewConfirm().
				// Inline(true).
				Title("Generate contract definition?").
				Description(" -- Generates contracts/definitions/"+name+"/ for in-process clients.").
				Value(&withContract),
			huh.NewConfirm().
				// Inline(true).
				Title("Is this module need database connection?").
				Description(" -- Generates cmd/"+name+"/migrate.go and migrations/ directory, etc.").
				Value(&withDatabase),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	root := platform.RootRepo()
	opts := scaffold.Options{
		Name:      name,
		RepoRoot:  root,
		OrgPrefix: detectOrgPrefix(root),
	}

	opts.WithConnect = withConnect
	opts.WithContract = withContract
	opts.WithDatabase = withDatabase

	if err := scaffold.GenerateModule(opts); err != nil {
		return fmt.Errorf("generate module: %w", err)
	}

	if err := scaffold.UpdateGoWork(opts.RepoRoot, name); err != nil {
		_, _ = errorC.Fprintf(os.Stderr, "warning: could not update go.work: %v\n", err)
	}

	if err := scaffold.UpdateMiseToml(opts.RepoRoot, name); err != nil {
		_, _ = errorC.Fprintf(os.Stderr, "warning: could not update mise.toml: %v\n", err)
	}

	successC.Printf("\n✓ Module %q generated in modules/%s/\n", name, name)
	warnC.Println("Next steps:")
	infoC.Printf("  - cd modules/%s && go mod tidy\n", name)
	infoC.Println("  - cd <repo-root> && mise run stow && go work sync")

	if withContract {
		infoC.Println("  - cd contracts && buf generate")
	}

	infoC.Println("  - Wire the module in cmd/mmw/main.go")

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
