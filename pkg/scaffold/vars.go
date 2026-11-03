package scaffold

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/piprim/goplt"
)

// Detected are the variables the command derives from the workspace instead
// of asking for them.
type Detected struct {
	OrgPrefix     string // first two elements of the contracts module path, e.g. ovya.fr/rcv
	ContractsPath string // module path of contracts/go.mod
	PlatformPath  string // module path of mmw itself
}

// ShapeGateway is the slice shape that calls an external service.
const ShapeGateway = "gateway"

const defaultPlatformPath = "github.com/piprim/mmw"

// Detect reads contracts/go.mod under repoRoot and the build info of the
// running binary.
func Detect(repoRoot string) (Detected, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "contracts", "go.mod"))
	if err != nil {
		return Detected{}, fmt.Errorf("detect contracts path: read contracts/go.mod: %w", err)
	}

	var contractsPath string
	for line := range strings.SplitSeq(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			contractsPath = strings.TrimSpace(rest)

			break
		}
	}
	if contractsPath == "" {
		return Detected{}, errors.New("detect contracts path: no module line in contracts/go.mod")
	}

	parts := strings.SplitN(contractsPath, "/", 3)
	if len(parts) < 2 {
		return Detected{}, fmt.Errorf("detect org prefix: module path %q has fewer than two elements", contractsPath)
	}

	platformPath := defaultPlatformPath
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Path != "" {
		platformPath = info.Main.Path
	}

	return Detected{
		OrgPrefix:     parts[0] + "/" + parts[1],
		ContractsPath: contractsPath,
		PlatformPath:  platformPath,
	}, nil
}

// Apply stores the detected values in vars under the keys the templates use.
func (d Detected) Apply(vars map[string]any) {
	vars["OrgPrefix"] = d.OrgPrefix
	vars["ContractsPath"] = d.ContractsPath
	vars["PlatformPath"] = d.PlatformPath
	if name, ok := vars["Name"].(string); ok {
		vars["ModulePath"] = d.OrgPrefix + "/" + name
	}
}

// AllowedShapes lists the slice shapes a module with or without a database
// accepts.
func AllowedShapes(database bool) []string {
	if database {
		return []string{"aggregate", "read-model", ShapeGateway}
	}

	return []string{ShapeGateway}
}

// ApplyShapeDefault turns an unset Shape (absent, empty, or the choice list
// ParseVarFlags seeds) into the default for Database.
func ApplyShapeDefault(vars map[string]any) {
	if s, _ := vars["Shape"].(string); s != "" {
		return
	}
	if database, _ := vars["Database"].(bool); database {
		vars["Shape"] = "aggregate"
	} else {
		vars["Shape"] = ShapeGateway
	}
}

// ParseVarFlags seeds vars with the manifest defaults, then overlays each
// "key=value" flag. Keys are normalised like goplt does (name, Name,
// create-table, CreateTable all work); bool variables are parsed.
func ParseVarFlags(m *goplt.Manifest, flags []string) (map[string]any, error) {
	kinds := make(map[string]goplt.VariableKind, len(m.Variables))
	vars := make(map[string]any, len(m.Variables))
	for _, v := range m.Variables {
		kinds[v.Name] = v.Kind
		vars[v.Name] = v.Value
	}

	for _, flag := range flags {
		key, value, ok := strings.Cut(flag, "=")
		if !ok {
			return nil, fmt.Errorf("--var %q: expected key=value", flag)
		}

		name := goplt.NormalizeKey(key)
		kind, known := kinds[name]
		if !known {
			return nil, fmt.Errorf("--var %q: unknown variable %q", flag, key)
		}

		switch kind {
		case goplt.KindBool:
			b, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("--var %q: %s must be true or false", flag, key)
			}
			vars[name] = b
		default:
			vars[name] = value
		}
	}

	return vars, nil
}

// MissingRequired returns the names of required input variables that are
// still empty, in manifest order.
func MissingRequired(m *goplt.Manifest, vars map[string]any) []string {
	var missing []string
	for _, v := range m.Variables {
		if !v.Required {
			continue
		}
		if s, _ := vars[v.Name].(string); s == "" {
			missing = append(missing, v.Name)
		}
	}

	return missing
}

// identRe is the rule for name and entity: a Go package name, a Postgres
// schema and a proto package all at once, so one lower-case word.
var identRe = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// reservedEntities are identifiers the templates generate in the entity's
// package, import names they place next to the domain package, and locals
// (receivers, parameters, variables) declared in a scope that also refers to
// the entity package; an entity with one of these names would collide with
// them. The locals were listed by an AST scan of the four rendered shapes.
var reservedEntities = []string{
	// generated identifiers and imports
	"domain", "label", "snapshot", "deleted", "event", "events", "errors",
	"command", "query", "ports", "application", "mapper", "connect", "uuid", "time", "context",
	"testing", "testhelpers", "authctx", "boa", "eris", "pgx", "boamodel", "postgres",
	"gateway", "publisher", "sent", "http", "httptest", "json", "bytes", "io", "pfevents", "protojson",
	"wiring", "health", "names", "mount", "stub", "helpers", "platform", "fmt", "strings", "sync",
	"inproc", "migrations", "stdlib", "wait", "pgxpool", "protoreflect", "timestamppb",
	// Module fields next to the slice's field
	"relay", "server", "router", "logger",
	// locals next to an entity-package reference
	"a", "adapter", "c", "cases", "cmd", "created", "ctx", "dispatcher", "e", "err", "ev", "g",
	"got", "handler", "id", "in", "input", "item", "lbl", "m", "method", "name", "now", "ok", "p",
	"payload", "pool", "provider", "q", "r", "repo", "res", "result", "s", "send", "snap", "status",
	"svc", "t", "tests", "topic", "tt", "uow", "updated", "want",
}

// reservedSchemas are Postgres reserved words that cannot name the schema
// unquoted. ponytail: the common subset, not the full keyword list.
var reservedSchemas = []string{
	"user", "order", "group", "table", "check", "select", "default", "primary", "references",
	"constraint", "grant", "session", "schema", "column", "index", "limit", "offset", "where",
	"from", "into", "union",
}

// Validate checks the module-level variables and that the module does not
// exist, then the slice with ValidateSlice.
func Validate(repoRoot string, vars map[string]any) error {
	name, _ := vars["Name"].(string)

	if !identRe.MatchString(name) {
		return fmt.Errorf("name %q must be one lower-case word (a-z0-9, letter first)", name)
	}
	if token.IsKeyword(name) {
		return fmt.Errorf("name %q is a Go keyword", name)
	}
	if slices.Contains(reservedSchemas, name) {
		return fmt.Errorf("name %q is a Postgres reserved word (it is the schema name)", name)
	}

	for _, rel := range []string{
		filepath.Join("modules", name),
		filepath.Join("contracts", "proto", name),
		filepath.Join("contracts", "buf.gen."+name+".yaml"),
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err == nil {
			return fmt.Errorf("%s already exists", filepath.ToSlash(rel))
		}
	}

	return ValidateSlice(repoRoot, vars)
}

// ValidateSlice checks the entity, the shape and that the entity's domain
// package and proto file are not there yet.
func ValidateSlice(repoRoot string, vars map[string]any) error {
	name, _ := vars["Name"].(string)
	entity, _ := vars["Entity"].(string)

	if !identRe.MatchString(entity) {
		return fmt.Errorf("entity %q must be one lower-case word (a-z0-9, letter first)", entity)
	}
	if token.IsKeyword(entity) {
		return fmt.Errorf("entity %q is a Go keyword", entity)
	}
	if slices.Contains(reservedEntities, entity) || types.Universe.Lookup(entity) != nil {
		return fmt.Errorf("entity %q is reserved (it collides with a generated identifier)", entity)
	}

	ApplyShapeDefault(vars)
	database, _ := vars["Database"].(bool)
	shape, _ := vars["Shape"].(string)
	if !slices.Contains(AllowedShapes(database), shape) {
		return fmt.Errorf("shape %q is not allowed in module %q (database=%t; allowed: %s)",
			shape, name, database, strings.Join(AllowedShapes(database), ", "))
	}

	for _, rel := range []string{
		filepath.Join("modules", name, "internal", "domain", entity),
		filepath.Join("contracts", "proto", name, "v1", entity+".proto"),
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err == nil {
			return fmt.Errorf("%s already exists", filepath.ToSlash(rel))
		}
	}

	return nil
}

// Render loads the manifest from fsys and writes the module and contract
// files under repoRoot. goplt writes every file as 0600; the probe script
// must be executable.
func Render(fsys fs.FS, repoRoot string, vars map[string]any) error {
	m, err := goplt.LoadManifest(fsys)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	if err := goplt.NewGenerator().Generate(fsys, m, repoRoot, vars); err != nil {
		return fmt.Errorf("generate module: %w", err)
	}

	name, _ := vars["Name"].(string)
	probes := filepath.Join(repoRoot, "modules", name, "scripts", "arch-probes.sh")
	if err := os.Chmod(probes, 0o755); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("chmod %s: %w", probes, err)
	}

	return nil
}
