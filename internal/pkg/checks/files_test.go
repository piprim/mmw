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
	c := checks.NewFilesChecker()

	assert.Equal(t, "files", c.Name())
}

func TestFilesChecker_Check_CleanFile(t *testing.T) {
	path := writeTemp(t, "clean.txt", "hello world\n")
	c := checks.NewFilesChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.False(t, result.HasViolations())
}

func TestFilesChecker_Check_TrailingWhitespace(t *testing.T) {
	path := writeTemp(t, "trailing.txt", "hello   \nworld\n")
	c := checks.NewFilesChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.True(t, result.HasViolations())
	assert.Len(t, result.Violations, 1)
	assert.Equal(t, 1, result.Violations[0].Line)
	assert.Contains(t, result.Violations[0].Message, "trailing whitespace")
}

func TestFilesChecker_Check_MissingNewline(t *testing.T) {
	path := writeTemp(t, "nonewline.txt", "hello world")
	c := checks.NewFilesChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.True(t, result.HasViolations())
	assert.Len(t, result.Violations, 1)
	assert.Contains(t, result.Violations[0].Message, "missing newline")
}

func TestFilesChecker_Check_LargeFile(t *testing.T) {
	content := make([]byte, 513_000) // > 500 KB
	for i := range content {
		content[i] = 'x'
	}
	content = append(content, '\n')
	path := writeTemp(t, "large.bin", string(content))
	c := checks.NewFilesChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.True(t, result.HasViolations())
	assert.Contains(t, result.Violations[0].Message, "500 KB")
}

func TestFilesChecker_Fix_RemovesTrailingWhitespace(t *testing.T) {
	path := writeTemp(t, "fix.txt", "hello   \nworld\n")

	fixer, ok := checks.NewFilesChecker().(checks.Fixer)
	require.True(t, ok, "filesChecker must implement Fixer")

	err := fixer.Fix(t.Context(), []string{path})
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello\nworld\n", string(got))
}

func TestFilesChecker_Fix_AddsNewline(t *testing.T) {
	path := writeTemp(t, "addnl.txt", "no newline")

	fixer := checks.NewFilesChecker().(checks.Fixer)

	err := fixer.Fix(t.Context(), []string{path})
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "no newline\n", string(got))
}

// writeTemp creates a temporary file with the given content and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	return path
}
