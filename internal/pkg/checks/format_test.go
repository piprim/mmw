package checks_test

import (
	"testing"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatChecker_Name(t *testing.T) {
	assert.Equal(t, "format", checks.NewFormatChecker().Name())
}

func TestFormatChecker_Check(t *testing.T) {
	t.Run("already formatted file has no violations", func(t *testing.T) {
		src := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
		path := writeTemp(t, "clean.go", src)
		result, err := checks.NewFormatChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.False(t, result.HasViolations(), "already-formatted file should have no violations")
	})

	t.Run("unformatted file reports violation with file path", func(t *testing.T) {
		unformatted := "package main\n\n\nimport \"fmt\"\n\n\nfunc main() {\nfmt.Println(\"hello\")\n}\n"
		path := writeTemp(t, "dirty.go", unformatted)
		result, err := checks.NewFormatChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.True(t, result.HasViolations(), "unformatted file should report a violation")
		assert.Equal(t, path, result.Violations[0].File)
	})

	t.Run("skips non-Go files", func(t *testing.T) {
		path := writeTemp(t, "config.yaml", "key: val\n")
		result, err := checks.NewFormatChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.False(t, result.HasViolations(), "non-Go files must be skipped")
	})
}

func TestFormatChecker_Fix(t *testing.T) {
	t.Run("formats unformatted file in-place so re-check passes", func(t *testing.T) {
		unformatted := "package main\n\n\nimport \"fmt\"\n\n\nfunc main() {\nfmt.Println(\"hello\")\n}\n"
		path := writeTemp(t, "fix.go", unformatted)

		fixer, ok := checks.NewFormatChecker().(checks.Fixer)
		require.True(t, ok, "formatChecker must implement Fixer")
		require.NoError(t, fixer.Fix(t.Context(), []string{path}))

		result, err := checks.NewFormatChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.False(t, result.HasViolations())
	})
}
