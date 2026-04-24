package custom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibDependencyValidator(t *testing.T) {
	t.Run("forbids imports from the services directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		libDir := filepath.Join(tmpDir, "libs", "mylib")
		os.MkdirAll(libDir, 0o755)
		os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/project\n\ngo 1.21\n"), 0o644)
		os.WriteFile(filepath.Join(libDir, "lib.go"), []byte(`package mylib

import (
	"github.com/test/project/services/todo"
)

func DoSomething() {}
`), 0o644)

		validator := &LibDependencyValidator{
			LibsDir:  filepath.Join(tmpDir, "libs"),
			RepoRoot: tmpDir,
		}

		err := validator.Check()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "services/")
	})

	t.Run("allows stdlib and external package imports", func(t *testing.T) {
		tmpDir := t.TempDir()
		libDir := filepath.Join(tmpDir, "libs", "mylib")
		os.MkdirAll(libDir, 0o755)
		os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/project\n\ngo 1.21\n"), 0o644)
		os.WriteFile(filepath.Join(libDir, "lib.go"), []byte(`package mylib

import (
	"context"
	"fmt"
	"github.com/external/package"
)

func DoSomething() {}
`), 0o644)

		validator := &LibDependencyValidator{
			LibsDir:  filepath.Join(tmpDir, "libs"),
			RepoRoot: tmpDir,
		}

		assert.NoError(t, validator.Check())
	})

	t.Run("allows importing other libs within the workspace", func(t *testing.T) {
		tmpDir := t.TempDir()
		libADir := filepath.Join(tmpDir, "libs", "liba")
		libBDir := filepath.Join(tmpDir, "libs", "libb")
		os.MkdirAll(libADir, 0o755)
		os.MkdirAll(libBDir, 0o755)
		os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module github.com/test/project\n\ngo 1.21\n"), 0o644)
		os.WriteFile(filepath.Join(libADir, "liba.go"), []byte("package liba\n\nfunc FuncA() {}\n"), 0o644)
		os.WriteFile(filepath.Join(libBDir, "libb.go"), []byte(`package libb

import (
	"github.com/test/project/libs/liba"
)

func FuncB() { liba.FuncA() }
`), 0o644)

		validator := &LibDependencyValidator{
			LibsDir:  filepath.Join(tmpDir, "libs"),
			RepoRoot: tmpDir,
		}

		assert.NoError(t, validator.Check())
	})

	t.Run("allows importing the mmw platform lib", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.work"), []byte(
			"go 1.26\nuse (\n\t./libs/mylib\n\t./mmw\n)\n",
		), 0600))

		mmwDir := filepath.Join(dir, "mmw")
		require.NoError(t, os.MkdirAll(mmwDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(mmwDir, "go.mod"), []byte(
			"module github.com/piprim/mmw\ngo 1.26\n",
		), 0600))

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
		assert.NoError(t, v.Check())
	})
}
