package custom

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// DomainPurityValidator checks that internal/domain/ never imports contracts/.
type DomainPurityValidator struct {
	ModulesDir string
}

func (v *DomainPurityValidator) Name() string { return "domain-purity" }

func (v *DomainPurityValidator) Description() string {
	return "internal/domain/ must not import contracts/ (transport concerns belong in adapters)"
}

func (v *DomainPurityValidator) Check() error {
	entries, err := os.ReadDir(v.ModulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read modules dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		domainDir := filepath.Join(v.ModulesDir, entry.Name(), "internal", "domain")
		if _, err := os.Stat(domainDir); os.IsNotExist(err) {
			continue
		}
		if err := walkForForbiddenImport(domainDir, "contracts", entry.Name(), "domain"); err != nil {
			return err
		}
	}
	return nil
}

// walkForForbiddenImport walks a directory and returns an error if any .go file
// imports a path containing the forbidden substring.
func walkForForbiddenImport(dir, forbidden, moduleName, layer string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			// Match forbidden as a complete path segment or segment component
			// e.g., "contracts" matches "/contracts/" and "/mmw-contracts/" but not "/my-contracts-util"
			if strings.Contains(importPath, "/"+forbidden+"/") ||
				strings.HasSuffix(importPath, "/"+forbidden) ||
				strings.HasPrefix(importPath, forbidden+"/") ||
				strings.Contains(importPath, "-"+forbidden+"/") ||
				strings.Contains(importPath, "/"+forbidden+"-") ||
				strings.Contains(importPath, "-"+forbidden+"-") {
				return fmt.Errorf(
					"modules/%s/internal/%s: imports %q — %s layer must not depend on contracts",
					moduleName, layer, importPath, layer,
				)
			}
		}
		return nil
	})
}
