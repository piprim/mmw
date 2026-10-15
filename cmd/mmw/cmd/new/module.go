package new

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/fatih/color"
	"github.com/piprim/goplt"
	gopltui "github.com/piprim/goplt/tui"
	"github.com/piprim/mmw/pkg/platform"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/spf13/cobra"
)

var (
	successC = color.New(color.FgGreen)
	infoC    = color.New(color.FgBlue)
	warnC    = color.New(color.FgYellow)
)

func NewModuleCmd() *cobra.Command {
	var (
		templatePath string
		varFlags     []string
	)

	cmd := &cobra.Command{
		Use:   "module",
		Short: "Scaffold a new hexagonal module under modules/<name>",
		Long: `Renders modules/<name>/ with one slice (an aggregate with database=true, a
gateway with database=false), its proto under contracts/proto/<name>/v1,
registers the module in go.work and mise.toml, then runs buf:generate, go mod
tidy and go work sync.

With --var for every required variable (name, entity) the TUI is skipped:

  mmw new module --var name=facturation --var entity=facture --var create-table=false
  mmw new module --var name=messagerie --var entity=sms --var database=false --var grpc=false`,
		RunE: func(c *cobra.Command, _ []string) error {
			return runNewModule(c.Context(), templatePath, varFlags)
		},
	}
	cmd.Flags().StringVar(&templatePath, "template", "",
		"Path to an external template directory (default: embedded templates)")
	cmd.Flags().StringArrayVar(&varFlags, "var", nil, "Set a template variable as key=value (repeatable)")

	return cmd
}

func runNewModule(ctx context.Context, templatePath string, varFlags []string) error {
	root := platform.RootRepo()

	fsys, err := selectTemplateFS(templatePath)
	if err != nil {
		return err
	}

	m, err := goplt.LoadManifest(fsys)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	vars, err := collectVars(m, varFlags)
	if err != nil {
		return err
	}

	detected, err := scaffold.Detect(root)
	if err != nil {
		return fmt.Errorf("detect workspace: %w", err)
	}
	detected.Apply(vars)

	if err := scaffold.Validate(root, vars); err != nil {
		return fmt.Errorf("invalid variables: %w", err)
	}

	name, _ := vars["Name"].(string)
	database, _ := vars["Database"].(bool)
	grpc, _ := vars["Grpc"].(bool)

	if err := scaffold.Render(fsys, root, vars); err != nil {
		return fmt.Errorf("render: %w", err)
	}
	if err := scaffold.UpdateGoWork(root, name); err != nil {
		return fmt.Errorf("update go.work: %w", err)
	}
	if err := scaffold.UpdateMiseToml(root, name, database, grpc); err != nil {
		return fmt.Errorf("update mise.toml: %w", err)
	}
	if err := scaffold.PostGenerate(ctx, os.Stdout, os.Stderr, root, name); err != nil {
		warnC.Println(err)
		printNextSteps(name, vars)
		printTableNote(name, vars)

		return fmt.Errorf("post-generate: %w", err)
	}

	printNextSteps(name, vars)
	printTableNote(name, vars)

	return nil
}

// collectVars returns the variables from --var alone when every required one
// is given, and opens the TUI pre-filled with them otherwise.
func collectVars(m *goplt.Manifest, varFlags []string) (map[string]any, error) {
	given, err := scaffold.ParseVarFlags(m, varFlags)
	if err != nil {
		return nil, fmt.Errorf("--var: %w", err)
	}

	if len(scaffold.MissingRequired(m, given)) == 0 {
		return given, nil
	}

	for i := range m.Variables {
		if v, ok := given[m.Variables[i].Name]; ok {
			m.Variables[i].Value = v
		}
	}

	orderVars(m)

	vars, err := gopltui.CollectVars(m)
	if err != nil {
		return nil, fmt.Errorf("prompt cancelled: %w", err)
	}

	return vars, nil
}

// promptOrder lists the variables the TUI asks first; goplt sorts them
// alphabetically, which puts create-table before the name it refers to.
var promptOrder = []string{"Name", "Entity", "Database", "CreateTable", "Grpc"}

// orderVars moves the promptOrder variables to the front, keeping the others
// in their original order.
func orderVars(m *goplt.Manifest) {
	rank := func(v goplt.Variable) int {
		if i := slices.Index(promptOrder, v.Name); i >= 0 {
			return i
		}

		return len(promptOrder)
	}
	slices.SortStableFunc(m.Variables, func(a, b goplt.Variable) int { return rank(a) - rank(b) })
}

// printTableNote tells the user which table to create when create-table=false.
func printTableNote(name string, vars map[string]any) {
	database, _ := vars["Database"].(bool)
	create, _ := vars["CreateTable"].(bool)
	if !database || create {
		return
	}
	entity, _ := vars["Entity"].(string)
	warnC.Printf("Note: create-table=false; the module expects table %s.%s with columns id BIGSERIAL PRIMARY KEY, "+
		"label VARCHAR(200) NOT NULL CHECK (length(label) > 0), created_at/updated_at TIMESTAMPTZ NOT NULL DEFAULT now(); "+
		"test:integration, the health check and boa:generate need it.\n", name, entity)
}

func printNextSteps(name string, vars map[string]any) {
	database, _ := vars["Database"].(bool)
	grpc, _ := vars["Grpc"].(bool)

	successC.Printf("\n✓ Module %q generated in modules/%s/\n", name, name)
	warnC.Println("Next:")
	infoC.Println("  - mise run stow                              (scripts/common, .golangci.yml)")
	if database {
		infoC.Printf("  - cd modules/%s && mise run db:migrate:up\n", name)
	}
	infoC.Println("  - add to initModules in cmd/cpro/main.go:")
	fields := "EventBus: eventBus, Subscriber: rawBus,"
	if database {
		fields = "DBPool: dbPool, " + fields
	}
	if grpc {
		fields += "\n            AuthSvc: authModule.PrivateService(),"
	}
	infoC.Printf(`        %[1]sModule, err := %[1]s.New(ctx, %[1]s.Infrastructure{
            %[2]s
            Logger: logger.With("module", %[1]s.ModuleName),
        })
        if err != nil { return nil, err }
`, name, fields)
	infoC.Println("    and append it to the returned slice, after core.")
	infoC.Printf("    other modules receive %[1]sModule.Service() (the contracts/go/application/%[1]s interface)\n", name)
	infoC.Printf("  - cd modules/%s && mise run arch:check && mise run lint && mise run test:unit\n", name)
}
