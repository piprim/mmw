package checks_test

import (
	"testing"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatChecker_Name(t *testing.T) {
	c := checks.NewFormatChecker()

	assert.Equal(t, "format", c.Name())
}

func TestFormatChecker_Check_FormattedFile(t *testing.T) {
	// Already formatted by gofumpt (single blank line between decls, etc.)
	src := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
	path := writeTemp(t, "clean.go", src)
	c := checks.NewFormatChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.False(t, result.HasViolations(), "already-formatted file should have no violations")
}

func TestFormatChecker_Check_UnformattedFile(t *testing.T) {
	// Extra blank lines that gofumpt would remove.
	unformatted := "package main\n\n\nimport \"fmt\"\n\n\nfunc main() {\nfmt.Println(\"hello\")\n}\n"
	path := writeTemp(t, "dirty.go", unformatted)
	c := checks.NewFormatChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.True(t, result.HasViolations(), "unformatted file should report a violation")
	assert.Equal(t, path, result.Violations[0].File)
}

func TestFormatChecker_Check_SkipsNonGoFiles(t *testing.T) {
	path := writeTemp(t, "config.yaml", "key: val\n")
	c := checks.NewFormatChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.False(t, result.HasViolations(), "non-Go files must be skipped")
}

func TestFormatChecker_Fix_FormatsFile(t *testing.T) {
	unformatted := "package main\n\n\nimport \"fmt\"\n\n\nfunc main() {\nfmt.Println(\"hello\")\n}\n"
	path := writeTemp(t, "fix.go", unformatted)

	fixer, ok := checks.NewFormatChecker().(checks.Fixer)
	require.True(t, ok, "formatChecker must implement Fixer")

	err := fixer.Fix(t.Context(), []string{path})
	require.NoError(t, err)

	// After fix, Check should report no violations.
	result, err := checks.NewFormatChecker().Check(t.Context(), []string{path})
	require.NoError(t, err)
	assert.False(t, result.HasViolations())
}
