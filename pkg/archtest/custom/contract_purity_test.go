package custom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractPurityValidator_ForbidsServiceModuleDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up a fake service module under modules/
	serviceDir := filepath.Join(tmpDir, "modules", "myservice")
	os.MkdirAll(serviceDir, 0o755)
	os.WriteFile(filepath.Join(serviceDir, "go.mod"), []byte("module github.com/test/myservice\n\ngo 1.21\n"), 0o644)

	// Set up go.work listing the service module (single-line use directives,
	// as the parser only handles "use <path>" not multi-line "use (...)" blocks)
	goWork := "go 1.21\n\nuse ./modules/myservice\nuse ./contracts/definitions/testcontract\n"
	os.WriteFile(filepath.Join(tmpDir, "go.work"), []byte(goWork), 0o644)

	// Set up contract that depends on the service module
	contractDir := filepath.Join(tmpDir, "contracts", "definitions", "testcontract")
	os.MkdirAll(contractDir, 0o755)
	goMod := "module github.com/test/testcontract\n\ngo 1.21\n\nrequire (\n\tgithub.com/test/myservice v0.0.0\n)\n"
	os.WriteFile(filepath.Join(contractDir, "go.mod"), []byte(goMod), 0o644)

	validator := &ContractPurityValidator{
		ContractsDir: filepath.Join(tmpDir, "contracts", "definitions"),
		RepoRoot:     tmpDir,
	}

	err := validator.Check()
	if err == nil {
		t.Error("Expected error for contract depending on a service module")
	}

	if !strings.Contains(err.Error(), "github.com/test/myservice") {
		t.Errorf("Expected error mentioning the forbidden service module, got: %v", err)
	}
}

func TestContractPurityValidator_NoInternalImports(t *testing.T) {
	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, "contracts", "definitions", "testcontract")
	os.MkdirAll(contractDir, 0o755)

	// Create go.mod with no dependencies
	goMod := `module github.com/test/testcontract

go 1.21
`
	os.WriteFile(filepath.Join(contractDir, "go.mod"), []byte(goMod), 0o644)

	// Create .go file with internal import
	goFile := `package testcontract

import (
	"github.com/test/services/todo/internal/domain"
)

type Client struct {}
`
	os.WriteFile(filepath.Join(contractDir, "client.go"), []byte(goFile), 0o644)

	validator := &ContractPurityValidator{
		ContractsDir: filepath.Join(tmpDir, "contracts", "definitions"),
	}

	err := validator.Check()
	if err == nil {
		t.Error("Expected error for contract importing internal package")
	}

	if !strings.Contains(err.Error(), "/internal/") {
		t.Errorf("Expected error about internal import, got: %v", err)
	}
}

func TestContractPurityValidator_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	contractDir := filepath.Join(tmpDir, "contracts", "definitions", "testcontract")
	os.MkdirAll(contractDir, 0o755)

	// Create clean go.mod
	goMod := `module github.com/test/testcontract

go 1.21
`
	os.WriteFile(filepath.Join(contractDir, "go.mod"), []byte(goMod), 0o644)

	// Create .go file with only stdlib imports
	goFile := `package testcontract

import (
	"context"
	"errors"
)

type Client struct {}
`
	os.WriteFile(filepath.Join(contractDir, "client.go"), []byte(goFile), 0o644)

	validator := &ContractPurityValidator{
		ContractsDir: filepath.Join(tmpDir, "contracts", "definitions"),
	}

	err := validator.Check()
	if err != nil {
		t.Errorf("Expected no error for valid contract, got: %v", err)
	}
}
