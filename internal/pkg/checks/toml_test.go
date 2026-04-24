package checks_test

import (
	"testing"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOMLChecker_Name(t *testing.T) {
	assert.Equal(t, "toml", checks.NewTOMLChecker().Name())
}

func TestTOMLChecker_Check(t *testing.T) {
	t.Run("valid TOML has no violations", func(t *testing.T) {
		path := writeTemp(t, "valid.toml", `
[section]
key = "value"
number = 42
`)
		result, err := checks.NewTOMLChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.False(t, result.HasViolations(), "valid TOML should have no violations")
	})

	t.Run("invalid TOML reports violation with file path", func(t *testing.T) {
		path := writeTemp(t, "invalid.toml", `
[section
key = "unclosed bracket"
`)
		result, err := checks.NewTOMLChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.True(t, result.HasViolations(), "invalid TOML should have violations")
		assert.Equal(t, path, result.Violations[0].File)
	})

	t.Run("skips non-TOML files silently", func(t *testing.T) {
		valid := writeTemp(t, "config.toml", `key = "val"`)
		yaml := writeTemp(t, "config.yaml", `key: val`)
		result, err := checks.NewTOMLChecker().Check(t.Context(), []string{valid, yaml})
		require.NoError(t, err)
		assert.False(t, result.HasViolations())
	})
}
