package checks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesChecker_Name(t *testing.T) {
	assert.Equal(t, "files", checks.NewFilesChecker().Name())
}

func TestFilesChecker_Check(t *testing.T) {
	t.Run("clean file has no violations", func(t *testing.T) {
		path := writeTemp(t, "clean.txt", "hello world\n")
		result, err := checks.NewFilesChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.False(t, result.HasViolations())
	})

	t.Run("reports trailing whitespace with line number", func(t *testing.T) {
		path := writeTemp(t, "trailing.txt", "hello   \nworld\n")
		result, err := checks.NewFilesChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.True(t, result.HasViolations())
		assert.Len(t, result.Violations, 1)
		assert.Equal(t, 1, result.Violations[0].Line)
		assert.Contains(t, result.Violations[0].Message, "trailing whitespace")
	})

	t.Run("reports missing newline at end of file", func(t *testing.T) {
		path := writeTemp(t, "nonewline.txt", "hello world")
		result, err := checks.NewFilesChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.True(t, result.HasViolations())
		assert.Len(t, result.Violations, 1)
		assert.Contains(t, result.Violations[0].Message, "missing newline")
	})

	t.Run("reports file exceeding 500 KB limit", func(t *testing.T) {
		content := make([]byte, 513_000)
		for i := range content {
			content[i] = 'x'
		}
		content = append(content, '\n')
		path := writeTemp(t, "large.bin", string(content))
		result, err := checks.NewFilesChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.True(t, result.HasViolations())
		assert.Contains(t, result.Violations[0].Message, "500 KB")
	})
}

func TestFilesChecker_Fix(t *testing.T) {
	fixer, ok := checks.NewFilesChecker().(checks.Fixer)
	require.True(t, ok, "filesChecker must implement Fixer")

	t.Run("strips trailing whitespace in-place", func(t *testing.T) {
		path := writeTemp(t, "fix.txt", "hello   \nworld\n")
		require.NoError(t, fixer.Fix(t.Context(), []string{path}))
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "hello\nworld\n", string(got))
	})

	t.Run("adds missing newline at end of file", func(t *testing.T) {
		path := writeTemp(t, "addnl.txt", "no newline")
		require.NoError(t, fixer.Fix(t.Context(), []string{path}))
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "no newline\n", string(got))
	})
}

// writeTemp creates a temporary file with the given content and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	return path
}
