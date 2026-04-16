package custom

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type LibDependencyValidator struct {
	LibsDir  string
	MmwDir   string // chemin vers mmw/ (lib platform, traitée comme lib)
	RepoRoot string
}

func (*LibDependencyValidator) Name() string {
	return "lib-dependency-purity"
}

func (*LibDependencyValidator) Description() string {
	return "libs/ packages can only import stdlib, external deps, or other libs"
}

func (v *LibDependencyValidator) Check() error {
	if _, err := os.Stat(v.LibsDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(v.LibsDir)
	if err != nil {
		return fmt.Errorf("failed to read libs directory: %w", err)
	}

	// Collect the module names of all libs (imports from these are allowed)
	libModules := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name, err := readModuleName(filepath.Join(v.LibsDir, entry.Name(), "go.mod"))
		if err == nil && name != "" {
			libModules[name] = true
		}
	}

	// Collect forbidden module names: any workspace module that is NOT a lib.
	// These are discovered from the go.work file.
	forbiddenModules, err := v.collectNonLibWorkspaceModules(libModules)
	if err != nil {
		// No go.work or unreadable — fall back to root module name check
		return v.checkWithRootModule(entries)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		libPath := filepath.Join(v.LibsDir, entry.Name())
		if err := checkLibImports(libPath, libModules, forbiddenModules); err != nil {
			return err
		}
	}

	return nil
}

// collectNonLibWorkspaceModules reads go.work and returns module names whose
// paths are NOT under the libs directory.
func (v *LibDependencyValidator) collectNonLibWorkspaceModules(libModules map[string]bool) (map[string]bool, error) {
	goWorkPath := filepath.Join(v.RepoRoot, "go.work")
	f, err := os.Open(goWorkPath)
	if err != nil {
		return nil, fmt.Errorf("open go.work: %w", err)
	}
	defer f.Close()

	absLibsDir, err := filepath.Abs(v.LibsDir)
	if err != nil {
		return nil, fmt.Errorf("abs libs dir: %w", err)
	}

	absMmwDir, err := v.absMMWDir()
	if err != nil {
		return nil, err
	}

	forbidden := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		v.scanUseLine(scanner.Text(), absLibsDir, absMmwDir, libModules, forbidden)
	}

	if err := scanner.Err(); err != nil {
		return forbidden, fmt.Errorf("scan go.work: %w", err)
	}

	return forbidden, nil
}

func (v *LibDependencyValidator) absMMWDir() (string, error) {
	if v.MmwDir == "" {
		return "", nil
	}

	abs, err := filepath.Abs(v.MmwDir)
	if err != nil {
		return "", fmt.Errorf("resolve MmwDir %q: %w", v.MmwDir, err)
	}

	return abs, nil
}

func (v *LibDependencyValidator) scanUseLine(
	rawLine, absLibsDir, absMmwDir string, libModules, forbidden map[string]bool,
) {
	line := strings.TrimSpace(rawLine)
	if !strings.HasPrefix(line, "use ") {
		return
	}

	relPath := strings.TrimPrefix(line, "use ")
	relPath = strings.Trim(relPath, `"`)

	absPath, err := filepath.Abs(filepath.Join(v.RepoRoot, relPath))
	if err != nil {
		return
	}

	sep := string(filepath.Separator)
	if strings.HasPrefix(absPath+sep, absLibsDir+sep) {
		return
	}

	if absMmwDir != "" && strings.HasPrefix(absPath+sep, absMmwDir+sep) {
		return
	}

	name, err := readModuleName(filepath.Join(absPath, "go.mod"))
	if err != nil || name == "" {
		return
	}

	if !libModules[name] {
		forbidden[name] = true
	}
}

// checkWithRootModule is the fallback when go.work is not available.
func (v *LibDependencyValidator) checkWithRootModule(entries []os.DirEntry) error {
	rootModuleName, err := readModuleName(filepath.Join(v.RepoRoot, "go.mod"))
	if err != nil {
		return fmt.Errorf("failed to get root module name: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		libPath := filepath.Join(v.LibsDir, entry.Name())
		if err := filepath.Walk(libPath, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			return checkFileAgainstRootModule(path, rootModuleName)
		}); err != nil {
			return fmt.Errorf("walk lib %q: %w", libPath, err)
		}
	}

	return nil
}

func checkLibImports(libPath string, _, forbiddenModules map[string]bool) error {
	if err := filepath.Walk(libPath, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %q: %w", path, err)
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for forbiddenModule := range forbiddenModules {
				if importPath == forbiddenModule || strings.HasPrefix(importPath, forbiddenModule+"/") {
					return fmt.Errorf(
						"%s: lib imports forbidden package: %s\n\n"+
							"libs/ packages can only import:\n"+
							"  - Standard library\n"+
							"  - External dependencies\n"+
							"  - Other libs/ packages\n\n"+
							"Forbidden: services/, contracts/, tools/, or root packages",
						path, importPath,
					)
				}
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walk lib %q: %w", libPath, err)
	}

	return nil
}

func checkFileAgainstRootModule(path, rootModuleName string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return fmt.Errorf("parse %q: %w", path, err)
	}

	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(importPath, rootModuleName+"/") {
			continue
		}
		relPath := strings.TrimPrefix(importPath, rootModuleName+"/")
		firstPart := strings.Split(relPath, "/")[0]
		if firstPart != "libs" {
			return fmt.Errorf(
				"%s: lib imports forbidden package: %s\n\n"+
					"libs/ packages can only import:\n"+
					"  - Standard library\n"+
					"  - External dependencies\n"+
					"  - Other libs/ packages\n\n"+
					"Forbidden: services/, contracts/, tools/, or root packages",
				path, importPath,
			)
		}
	}

	return nil
}

func readModuleName(goModPath string) (string, error) {
	content, err := os.ReadFile(goModPath) //nolint:gosec // path is constructed from trusted workspace internals
	if err != nil {
		return "", fmt.Errorf("read go.mod %q: %w", goModPath, err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("module name not found in %s", goModPath)
}
