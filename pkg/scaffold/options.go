package scaffold

import (
	"fmt"
	"regexp"
	"strings"
)

// validModuleName matches lowercase alphanumeric names with optional hyphens.
var validModuleName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Options holds the user's choices from the interactive CLI.
type Options struct {
	Name          string // "payment" — lowercase, no spaces
	OrgPrefix     string // "github.com/acme" — used to build module paths
	RepoRoot      string // absolute path to poc/ root
	WithConnect   bool   // generate Connect RPC handler + proto
	WithContract  bool   // generate contracts/definitions/<name>/
	WithDatabase bool   // generate migration command + scripts dir
}

// ModuleData is the template data passed to every template file.
type ModuleData struct {
	Name          string // "payment"
	NameTitle     string // "Payment"
	ModulePath    string // "github.com/acme/mmw-payment"
	ContractsPath string // "github.com/acme/mmw-contracts"
	PlatformPath  string // "github.com/piprim/mmw"
	PkgDef        string // "defpayment" — alias for contracts/definitions/payment import
	WithConnect   bool
	WithContract  bool
	WithDatabase bool
}

// newModuleData builds template data from Options.
func newModuleData(opts Options) (*ModuleData, error) {
	if !validModuleName.MatchString(opts.Name) {
		return &ModuleData{}, fmt.Errorf("scaffold: Name must match [a-z][a-z0-9-]* (got %q)", opts.Name)
	}
	title := strings.ToUpper(opts.Name[:1]) + opts.Name[1:]

	return &ModuleData{
		Name:          opts.Name,
		NameTitle:     title,
		ModulePath:    opts.OrgPrefix + "/mmw-" + opts.Name,
		ContractsPath: opts.OrgPrefix + "/mmw-contracts",
		PlatformPath:  "github.com/piprim/mmw",
		PkgDef:        "def" + opts.Name,
		WithConnect:   opts.WithConnect,
		WithContract:  opts.WithContract,
		WithDatabase: opts.WithDatabase,
	}, nil
}
