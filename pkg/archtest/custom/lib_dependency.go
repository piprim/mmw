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
		return nil, err
	}
	defer f.Close()

	absLibsDir, err := filepath.Abs(v.LibsDir)
	if err != nil {
		return nil, err
	}

	absMmwDir := ""
	if v.MmwDir != "" {
		abs, err := filepath.Abs(v.MmwDir)
		if err != nil {
			return nil, fmt.Errorf("resolve MmwDir %q: %w", v.MmwDir, err)
		}
		absMmwDir = abs
	}

	forbidden := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "use ") {
			continue
		}
		relPath := strings.TrimPrefix(line, "use ")
		relPath = strings.Trim(relPath, `"`)
		absPath, err := filepath.Abs(filepath.Join(v.RepoRoot, relPath))
		if err != nil {
			continue
		}

		// Skip if this module lives under the libs directory
		if strings.HasPrefix(absPath+string(filepath.Separator), absLibsDir+string(filepath.Separator)) {
			continue
		}

		// Skip if this module lives under mmw/ (treated as a lib)
		if absMmwDir != "" && strings.HasPrefix(absPath+string(filepath.Separator), absMmwDir+string(filepath.Separator)) {
			continue
		}

		name, err := readModuleName(filepath.Join(absPath, "go.mod"))
		if err != nil || name == "" {
			continue
		}
		if !libModules[name] {
			forbidden[name] = true
		}
	}

	return forbidden, scanner.Err()
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
		err := filepath.Walk(libPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			return checkFileAgainstRootModule(path, rootModuleName)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func checkLibImports(libPath string, libModules, forbiddenModules map[string]bool) error {
	return filepath.Walk(libPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
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
	})
}

func checkFileAgainstRootModule(path, rootModuleName string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}

	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if strings.HasPrefix(importPath, rootModuleName+"/") {
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
	}

	return nil
}

func readModuleName(goModPath string) (string, error) {
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("module name not found in %s", goModPath)
}
