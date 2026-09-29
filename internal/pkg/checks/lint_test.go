package checks_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubGolangciLint puts a stub golangci-lint on PATH that records its arguments.
// The returned func reports them, or "" when the stub was never invoked.
func stubGolangciLint(t *testing.T) func() string {
	t.Helper()

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := "#!/bin/sh\necho \"$@\" > '" + argsFile + "'\n"

	//nolint:gosec // the stub must be executable
	require.NoError(t, os.WriteFile(filepath.Join(dir, "golangci-lint"), []byte(script), 0o700))
	t.Setenv("PATH", dir)

	return func() string {
		data, err := os.ReadFile(argsFile)
		if errors.Is(err, fs.ErrNotExist) {
			return ""
		}

		require.NoError(t, err)

		return strings.TrimSpace(string(data))
	}
}

func TestLintFilesChecker(t *testing.T) {
	tests := []struct {
		name     string
		targets  []string
		wantArgs string // "" = golangci-lint must not run
	}{
		{
			name:     "submodule path is not linted as a package",
			targets:  []string{"mmw"},
			wantArgs: "",
		},
		{
			name:     "extensionless file is not linted as a package",
			targets:  []string{"Makefile", "LICENSE"},
			wantArgs: "",
		},
		{
			name:     "go files are linted by package directory",
			targets:  []string{"mmw", "cmd/root.go", "internal/pkg/checks/lint.go", "go.mod"},
			wantArgs: "run cmd internal/pkg/checks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs := stubGolangciLint(t)

			result, err := checks.NewLintFilesChecker(io.Discard, io.Discard).Check(t.Context(), tt.targets)
			require.NoError(t, err)
			assert.False(t, result.HasViolations())
			assert.Equal(t, tt.wantArgs, gotArgs())
		})
	}
}

func TestLintChecker(t *testing.T) {
	t.Run("extensionless package pattern is passed through", func(t *testing.T) {
		gotArgs := stubGolangciLint(t)

		result, err := checks.NewLintChecker(io.Discard, io.Discard).Check(t.Context(), []string{"internal/pkg/checks"})
		require.NoError(t, err)
		assert.False(t, result.HasViolations())
		assert.Equal(t, "run internal/pkg/checks", gotArgs())
	})
}
