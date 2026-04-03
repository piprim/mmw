package custom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibDependencyValidator_NoRootDirImports(t *testing.T) {
	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "libs", "mylib")
	os.MkdirAll(libDir, 0o755)

	// Create root go.mod
	rootGoMod := `module github.com/test/project

go 1.21
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(rootGoMod), 0o644)

	// Create lib file importing services (forbidden)
	libFile := `package mylib

import (
	"github.com/test/project/services/todo"
)

func DoSomething() {}
`
	os.WriteFile(filepath.Join(libDir, "lib.go"), []byte(libFile), 0o644)

	validator := &LibDependencyValidator{
		LibsDir:  filepath.Join(tmpDir, "libs"),
		RepoRoot: tmpDir,
	}

	err := validator.Check()
	if err == nil {
		t.Error("Expected error for lib importing services/")
	}

	if !strings.Contains(err.Error(), "services/") {
		t.Errorf("Expected error about services/ import, got: %v", err)
	}
}

func TestLibDependencyValidator_AllowsStdlibAndExternal(t *testing.T) {
	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "libs", "mylib")
	os.MkdirAll(libDir, 0o755)

	// Create root go.mod
	rootGoMod := `module github.com/test/project

go 1.21
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(rootGoMod), 0o644)

	// Create lib file importing stdlib and external deps
	libFile := `package mylib

import (
	"context"
	"fmt"
	"github.com/external/package"
)

func DoSomething() {}
`
	os.WriteFile(filepath.Join(libDir, "lib.go"), []byte(libFile), 0o644)

	validator := &LibDependencyValidator{
		LibsDir:  filepath.Join(tmpDir, "libs"),
		RepoRoot: tmpDir,
	}

	err := validator.Check()
	if err != nil {
		t.Errorf("Expected no error for stdlib and external imports, got: %v", err)
	}
}

func TestLibDependencyValidator_AllowsOtherLibs(t *testing.T) {
	tmpDir := t.TempDir()
	libADir := filepath.Join(tmpDir, "libs", "liba")
	libBDir := filepath.Join(tmpDir, "libs", "libb")
	os.MkdirAll(libADir, 0o755)
	os.MkdirAll(libBDir, 0o755)

	// Create root go.mod
	rootGoMod := `module github.com/test/project

go 1.21
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(rootGoMod), 0o644)

	// Create lib A
	libAFile := `package liba

func FuncA() {}
`
	os.WriteFile(filepath.Join(libADir, "liba.go"), []byte(libAFile), 0o644)

	// Create lib B importing lib A (allowed)
	libBFile := `package libb

import (
	"github.com/test/project/libs/liba"
)

func FuncB() {
	liba.FuncA()
}
`
	os.WriteFile(filepath.Join(libBDir, "libb.go"), []byte(libBFile), 0o644)

	validator := &LibDependencyValidator{
		LibsDir:  filepath.Join(tmpDir, "libs"),
		RepoRoot: tmpDir,
	}

	err := validator.Check()
	if err != nil {
		t.Errorf("Expected no error for lib importing other lib, got: %v", err)
	}
}

func TestLibDependencyValidator_MmwDir_IsAllowed(t *testing.T) {
	// A lib should be allowed to import mmw (platform lib) without being flagged.
	dir := t.TempDir()

	// Create fake go.work that references mmw/ as a workspace module
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.work"), []byte(
		"go 1.26\nuse (\n\t./libs/mylib\n\t./mmw\n)\n",
	), 0600))

	// Create mmw go.mod
	mmwDir := filepath.Join(dir, "mmw")
	require.NoError(t, os.MkdirAll(mmwDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(mmwDir, "go.mod"), []byte(
		"module github.com/piprim/mmw\ngo 1.26\n",
	), 0600))

	// Create a lib that imports mmw — should be allowed
	libDir := filepath.Join(dir, "libs", "mylib")
	require.NoError(t, os.MkdirAll(libDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "go.mod"), []byte(
		"module github.com/acme/mylib\ngo 1.26\n",
	), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "util.go"), []byte(
		"package mylib\nimport _ \"github.com/piprim/mmw\"\n",
	), 0600))

	v := &LibDependencyValidator{
		LibsDir:  filepath.Join(dir, "libs"),
		MmwDir:   filepath.Join(dir, "mmw"),
		RepoRoot: dir,
	}
	// Should pass: mmw is treated as a lib, import is allowed
	assert.NoError(t, v.Check())
}
