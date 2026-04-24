package custom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractPurityValidator(t *testing.T) {
	t.Run("forbids contract depending on a service module", func(t *testing.T) {
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
			t.Error("expected error for contract depending on a service module")
		}
		if !strings.Contains(err.Error(), "github.com/test/myservice") {
			t.Errorf("expected error mentioning the forbidden service module, got: %v", err)
		}
	})

	t.Run("forbids internal package imports from contracts", func(t *testing.T) {
		tmpDir := t.TempDir()
		contractDir := filepath.Join(tmpDir, "contracts", "definitions", "testcontract")
		os.MkdirAll(contractDir, 0o755)

		os.WriteFile(filepath.Join(contractDir, "go.mod"), []byte("module github.com/test/testcontract\n\ngo 1.21\n"), 0o644)
		os.WriteFile(filepath.Join(contractDir, "client.go"), []byte(`package testcontract

import (
	"github.com/test/services/todo/internal/domain"
)

type Client struct {}
`), 0o644)

		validator := &ContractPurityValidator{
			ContractsDir: filepath.Join(tmpDir, "contracts", "definitions"),
		}

		err := validator.Check()
		if err == nil {
			t.Error("expected error for contract importing internal package")
		}
		if !strings.Contains(err.Error(), "/internal/") {
			t.Errorf("expected error about internal import, got: %v", err)
		}
	})

	t.Run("passes for valid contract with only stdlib imports", func(t *testing.T) {
		tmpDir := t.TempDir()
		contractDir := filepath.Join(tmpDir, "contracts", "definitions", "testcontract")
		os.MkdirAll(contractDir, 0o755)

		os.WriteFile(filepath.Join(contractDir, "go.mod"), []byte("module github.com/test/testcontract\n\ngo 1.21\n"), 0o644)
		os.WriteFile(filepath.Join(contractDir, "client.go"), []byte(`package testcontract

import (
	"context"
	"errors"
)

type Client struct {}
`), 0o644)

		validator := &ContractPurityValidator{
			ContractsDir: filepath.Join(tmpDir, "contracts", "definitions"),
		}

		if err := validator.Check(); err != nil {
			t.Errorf("expected no error for valid contract, got: %v", err)
		}
	})
}
