package custom

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type ContractPurityValidator struct {
	ContractsDir string
	RepoRoot     string
}

func (v *ContractPurityValidator) Name() string {
	return "contract-definition-purity"
}

func (v *ContractPurityValidator) Description() string {
	return "Contract definition modules must not depend on service modules or import internal packages"
}

func (v *ContractPurityValidator) Check() error {
	if _, err := os.Stat(v.ContractsDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(v.ContractsDir)
	if err != nil {
		return fmt.Errorf("failed to read contracts directory: %w", err)
	}

	// Collect service module names from the workspace — these are forbidden
	// dependencies for contract definition modules.
	forbiddenModules, err := v.collectServiceModules()
	if err != nil {
		// If go.work is unavailable, skip the go.mod check
		forbiddenModules = nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		contractPath := filepath.Join(v.ContractsDir, entry.Name())

		if err := v.checkGoModPurity(contractPath, entry.Name(), forbiddenModules); err != nil {
			return err
		}

		if err := v.checkNoInternalImports(contractPath, entry.Name()); err != nil {
			return err
		}
	}

	return nil
}

// collectServiceModules reads go.work and returns module names whose paths
// are under the services/ directory (i.e., not libs or contracts or tools).
func (v *ContractPurityValidator) collectServiceModules() (map[string]bool, error) {
	if v.RepoRoot == "" {
		return nil, nil
	}

	goWorkPath := filepath.Join(v.RepoRoot, "go.work")
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		return nil, err
	}

	absServicesDir, _ := filepath.Abs(filepath.Join(v.RepoRoot, "modules"))
	forbidden := make(map[string]bool)

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "use ") {
			continue
		}
		relPath := strings.Trim(strings.TrimPrefix(line, "use "), `"`)
		absPath, err := filepath.Abs(filepath.Join(v.RepoRoot, relPath))
		if err != nil {
			continue
		}
		if !strings.HasPrefix(absPath+string(filepath.Separator), absServicesDir+string(filepath.Separator)) {
			continue
		}
		name, err := readModuleName(filepath.Join(absPath, "go.mod"))
		if err == nil && name != "" {
			forbidden[name] = true
		}
	}

	return forbidden, nil
}

// checkGoModPurity verifies the contract's go.mod has no dependencies on
// service modules. Protocol/transport libraries (connectrpc, protobuf, etc.)
// are allowed — they are part of the API contract definition.
func (v *ContractPurityValidator) checkGoModPurity(contractPath, contractName string, forbiddenModules map[string]bool) error {
	if len(forbiddenModules) == 0 {
		return nil
	}

	goModPath := filepath.Join(contractPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire || strings.HasPrefix(line, "require ") {
			if strings.Contains(line, "// indirect") || strings.HasPrefix(line, "//") {
				continue
			}

			parts := strings.Fields(line)
			if len(parts) == 0 {
				continue
			}
			modulePath := parts[0]

			if forbiddenModules[modulePath] {
				return fmt.Errorf(
					"contracts/definitions/%s depends on service module: %s\n\n"+
						"Contract definition modules must not depend on service modules.\n"+
						"They may only depend on:\n"+
						"  - Standard library\n"+
						"  - Protocol/transport libraries (connectrpc, protobuf, grpc, etc.)\n"+
						"  - Basic utility types (uuid, etc.)\n"+
						"  - Other contract definition modules",
					contractName, modulePath,
				)
			}
		}
	}

	return nil
}

func (v *ContractPurityValidator) checkNoInternalImports(contractPath, contractName string) error {
	return filepath.Walk(contractPath, func(path string, info os.FileInfo, err error) error {
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

			if strings.Contains(importPath, "/internal/") {
				return fmt.Errorf(
					"contracts/definitions/%s/%s: imports internal package: %s\n\n"+
						"Contract definition modules must NEVER import internal/ packages from any service.\n\n"+
						"If this is InprocServer, it belongs in the service's internal adapters:\n"+
						"  Move to: services/%s/internal/adapters/inbound/contracts/inproc_server.go\n\n"+
						"Contract definition modules can only import:\n"+
						"  - Standard library\n"+
						"  - Other contract definition modules (public APIs)",
					contractName, filepath.Base(path), importPath, contractName,
				)
			}
		}

		return nil
	})
}
