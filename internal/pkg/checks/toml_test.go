package checks_test

import (
	"testing"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOMLChecker_Name(t *testing.T) {
	c := checks.NewTOMLChecker()

	assert.Equal(t, "toml", c.Name())
}

func TestTOMLChecker_Check_ValidTOML(t *testing.T) {
	path := writeTemp(t, "valid.toml", `
[section]
key = "value"
number = 42
`)
	c := checks.NewTOMLChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.False(t, result.HasViolations(), "valid TOML should have no violations")
}

func TestTOMLChecker_Check_InvalidTOML(t *testing.T) {
	path := writeTemp(t, "invalid.toml", `
[section
key = "unclosed bracket"
`)
	c := checks.NewTOMLChecker()

	result, err := c.Check(t.Context(), []string{path})

	require.NoError(t, err)
	assert.True(t, result.HasViolations(), "invalid TOML should have violations")
	assert.Equal(t, path, result.Violations[0].File)
}

func TestTOMLChecker_Check_FiltersTOMLOnly(t *testing.T) {
	valid := writeTemp(t, "config.toml", `key = "val"`)
	yaml := writeTemp(t, "config.yaml", `key: val`)
	c := checks.NewTOMLChecker()

	// YAML file should be silently skipped; only .toml is checked.
	result, err := c.Check(t.Context(), []string{valid, yaml})

	require.NoError(t, err)
	assert.False(t, result.HasViolations())
}
