package scaffold

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:_templates
var templatesFS embed.FS

// EmbeddedFS returns the embedded templates as an fs.FS with the "_templates/" prefix stripped.
func EmbeddedFS() fs.FS {
	sub, err := fs.Sub(templatesFS, "_templates")
	if err != nil {
		panic(fmt.Sprintf("scaffold: embedded templates FS error: %v", err))
	}

	return sub
}

// GenerateModule generates a complete module (and contracts when WithContract=true)
// by walking fsys and writing rendered files to repoRoot.
func GenerateModule(fsys fs.FS, repoRoot string, vars map[string]any) error {
	return generate(fsys, repoRoot, vars, ".")
}

// GenerateContract generates only the contract definition files.
func GenerateContract(fsys fs.FS, repoRoot string, vars map[string]any) error {
	return generate(fsys, repoRoot, vars, "contracts")
}

type generator struct {
	manifest *Manifest
	fsys     fs.FS
	repoRoot string
	vars     map[string]any
}

func generate(fsys fs.FS, repoRoot string, vars map[string]any, root string) error {
	if name, ok := vars["Name"].(string); !ok || name == "" {
		return errors.New("scaffold: Name is required")
	}
	m, err := LoadManifest(fsys)
	if err != nil {
		return err
	}
	g := &generator{manifest: m, fsys: fsys, repoRoot: repoRoot, vars: vars}

	if err := fs.WalkDir(fsys, root, g.walk); err != nil {
		return fmt.Errorf("walk template %q: %w", root, err)
	}

	return nil
}

func (g *generator) walk(path string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if path == "." || path == "template.toml" {
		return nil
	}

	skipped, err := g.isConditionedOut(path)
	if err != nil {
		return err
	}
	if skipped {
		if d.IsDir() {
			return fs.SkipDir
		}

		return nil
	}

	if d.IsDir() {
		return nil
	}

	outPath, err := renderString(strings.TrimSuffix(path, ".tmpl"), g.vars)
	if err != nil {
		return fmt.Errorf("render path %q: %w", path, err)
	}

	content, err := fs.ReadFile(g.fsys, path)
	if err != nil {
		return fmt.Errorf("read template %q: %w", path, err)
	}

	rendered, err := renderBytes(path, content, g.vars)
	if err != nil {
		return fmt.Errorf("render content of %q: %w", path, err)
	}

	absPath := filepath.Join(g.repoRoot, outPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("mkdir for %q: %w", absPath, err)
	}

	if err := os.WriteFile(absPath, rendered, 0600); err != nil {
		return fmt.Errorf("write %q: %w", absPath, err)
	}

	return nil
}

// isConditionedOut returns true if the file at path should be skipped
// because a condition in the manifest evaluated to empty/false.
func (g *generator) isConditionedOut(path string) (bool, error) {
	for prefix, expr := range g.manifest.Conditions {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		result, err := renderString(expr, g.vars)
		if err != nil {
			return false, fmt.Errorf("evaluate condition for prefix %q: %w", prefix, err)
		}
		if result == "" {
			return true, nil
		}
	}

	return false, nil
}

// funcMap provides template helper functions.
func funcMap() template.FuncMap {
	return template.FuncMap{
		"config_root": func() string { return "{{config_root}}" },
	}
}

func renderString(tmplStr string, data any) (string, error) {
	t, err := template.New("").Funcs(funcMap()).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", tmplStr, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", tmplStr, err)
	}

	return buf.String(), nil
}

func renderBytes(name string, content []byte, data any) ([]byte, error) {
	t, err := template.New(name).Funcs(funcMap()).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %q: %w", name, err)
	}

	return buf.Bytes(), nil
}
