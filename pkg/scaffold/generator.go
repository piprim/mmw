package scaffold

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates
var templatesFS embed.FS

// fileSpec describes a file to generate.
type fileSpec struct {
	tmplPath   string                 // path inside templates/
	outputPath string                 // relative to repoRoot; may contain {{.Name}} etc.
	condition  func(*ModuleData) bool // nil = always generate
}

// moduleSpecs defines every file generated for a module.
var moduleSpecs = []fileSpec{
	// Base
	{tmplPath: "templates/module/go.mod.tmpl",
		outputPath: "modules/{{.Name}}/go.mod"},
	{tmplPath: "templates/module/mod.go.tmpl",
		outputPath: "modules/{{.Name}}/{{.Name}}mod.go"},
	{tmplPath: "templates/module/mise.toml.tmpl",
		outputPath: "modules/{{.Name}}/mise.toml"},
	// Domain
	{tmplPath: "templates/module/domain/aggregate.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/domain/{{.Name}}.go"},
	{tmplPath: "templates/module/domain/value_objects.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/domain/value_objects.go"},
	{tmplPath: "templates/module/domain/events.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/domain/events.go"},
	{tmplPath: "templates/module/domain/errors.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/domain/errors.go"},
	// Application
	{tmplPath: "templates/module/application/service.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/application/service.go"},
	{tmplPath: "templates/module/application/errors.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/application/errors.go"},
	{tmplPath: "templates/module/application/ports/ports.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/application/ports/ports.go"},
	// Adapters — outbound (always)
	{tmplPath: "templates/module/adapters/outbound/persistence/postgres/repository.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/adapters/outbound/persistence/postgres/repository.go"},
	{tmplPath: "templates/module/adapters/outbound/events/topics.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/adapters/outbound/events/topics.go"},
	{tmplPath: "templates/module/adapters/outbound/events/outbox_dispatcher.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/adapters/outbound/events/outbox_dispatcher.go"},
	// Infra — config (always)
	{tmplPath: "templates/module/infra/config/config.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/infra/config/config.go"},
	{tmplPath: "templates/module/infra/config/configs/default.toml.tmpl",
		outputPath: "modules/{{.Name}}/internal/infra/config/configs/default.toml"},
	{tmplPath: "templates/module/infra/config/configs/development.toml.tmpl",
		outputPath: "modules/{{.Name}}/internal/infra/config/configs/development.toml"},
	// Adapters — inbound connect (conditional)
	{
		tmplPath:   "templates/module/adapters/inbound/connect/handler.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/adapters/inbound/connect/handler.go",
		condition:  func(d *ModuleData) bool { return d.WithConnect },
	},
	{
		tmplPath:   "templates/module/adapters/inbound/connect/errors.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/adapters/inbound/connect/errors.go",
		condition:  func(d *ModuleData) bool { return d.WithConnect },
	},
	// Adapters — inbound inproc (conditional: only when Contract)
	{
		tmplPath:   "templates/module/adapters/inbound/inproc/adapter.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/adapters/inbound/inproc/adapter.go",
		condition:  func(d *ModuleData) bool { return d.WithContract },
	},
	// Infra — database (conditional)
	{
		tmplPath:   "templates/module/infra/migrations/migrations.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/infra/persistence/migrations/migrations.go",
		condition:  func(d *ModuleData) bool { return d.WithDatabase },
	},
	{
		tmplPath:   "templates/module/infra/migrations/scripts/00001_empty.go.tmpl",
		outputPath: "modules/{{.Name}}/internal/infra/persistence/migrations/scripts/00001_empty.go",
		condition:  func(d *ModuleData) bool { return d.WithDatabase },
	},
	{
		tmplPath:   "templates/module/infra/migrations/scripts/00002_empty.sql.tmpl",
		outputPath: "modules/{{.Name}}/internal/infra/persistence/migrations/scripts/00002_empty.sql",
		condition:  func(d *ModuleData) bool { return d.WithDatabase },
	},
	{
		tmplPath:   "templates/module/cmd/migrate/main.go.tmpl",
		outputPath: "modules/{{.Name}}/cmd/{{.Name}}/migrate.go",
		condition:  func(d *ModuleData) bool { return d.WithDatabase },
	},
}

// contractSpecs defines every file generated for a contract definition.
var contractSpecs = []fileSpec{
	{tmplPath: "templates/contract/api.go.tmpl", outputPath: "contracts/definitions/{{.Name}}/api.go"},
	{tmplPath: "templates/contract/dto.go.tmpl", outputPath: "contracts/definitions/{{.Name}}/dto.go"},
	{tmplPath: "templates/contract/errors.go.tmpl", outputPath: "contracts/definitions/{{.Name}}/errors.go"},
	{tmplPath: "templates/contract/inproc_client.go.tmpl", outputPath: "contracts/definitions/{{.Name}}/inproc_client.go"},
	{
		tmplPath:   "templates/contract/proto.proto.tmpl",
		outputPath: "contracts/proto/{{.Name}}/v1/{{.Name}}.proto",
		condition:  func(d *ModuleData) bool { return d.WithConnect },
	},
}

// GenerateModule generates all module files for the given options.
func GenerateModule(opts Options) error {
	data, err := newModuleData(opts)
	if err != nil {
		return err
	}
	specs := make([]fileSpec, len(moduleSpecs))
	copy(specs, moduleSpecs)
	if opts.WithContract {
		specs = append(specs, contractSpecs...)
	}

	return renderSpecs(specs, data, opts.RepoRoot)
}

// GenerateContract generates only the contract definition files.
func GenerateContract(opts Options) error {
	data, err := newModuleData(opts)
	if err != nil {
		return err
	}

	return renderSpecs(contractSpecs, data, opts.RepoRoot)
}

// renderSpecs executes all applicable fileSpecs.
func renderSpecs(specs []fileSpec, data *ModuleData, repoRoot string) error {
	for _, spec := range specs {
		if spec.condition != nil && !spec.condition(data) {
			continue
		}

		outPath, err := renderString(spec.outputPath, data)
		if err != nil {
			return fmt.Errorf("render output path %q: %w", spec.outputPath, err)
		}

		tmplContent, err := fs.ReadFile(templatesFS, spec.tmplPath)
		if err != nil {
			return fmt.Errorf("read template %q: %w", spec.tmplPath, err)
		}

		rendered, err := renderBytes(spec.tmplPath, tmplContent, data)
		if err != nil {
			return fmt.Errorf("render template %q: %w", spec.tmplPath, err)
		}

		absPath := filepath.Join(repoRoot, outPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return fmt.Errorf("mkdir %q: %w", filepath.Dir(absPath), err)
		}
		if err := os.WriteFile(absPath, rendered, 0600); err != nil {
			return fmt.Errorf("write %q: %w", absPath, err)
		}
	}

	return nil
}

// funcMap provides template functions, including Mise-specific ones.
// These are evaluated during generation and return literal Mise template syntax
// that will be evaluated by Mise at runtime.
func funcMap() template.FuncMap {
	return template.FuncMap{
		// config_root is a Mise-specific function; we emit it literally for Mise to evaluate
		"config_root": func() string { return "{{config_root}}" },
		// Additional Mise functions can be added here as needed
	}
}

func renderString(tmplStr string, data *ModuleData) (string, error) {
	t, err := template.New("").Funcs(funcMap()).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf(`failed to parse template '%s':
 %w`, tmplStr, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf(`failed to execute template '%s' with data '%v': %w`, tmplStr, data, err)
	}

	return buf.String(), nil
}

func renderBytes(name string, tmplContent []byte, data *ModuleData) ([]byte, error) {
	t, err := template.New(name).Funcs(funcMap()).Parse(string(tmplContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template '%s': %w", string(tmplContent), err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf(`failed to execute template '%s'
with data '%v':
%w`, string(tmplContent), *data, err)
	}

	return buf.Bytes(), nil
}
