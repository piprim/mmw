package checks_test

import (
	"os/exec"
	"testing"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// longLineYAML carries a comment past the 80-column mark. Under `-d relaxed`
// yamllint reports that as a warning and still exits 0, so it must not fail
// the check.
const longLineYAML = "---\n" +
	"# this is a deliberately long comment line that goes well past the eighty character limit\n" +
	"key: value\n"

func requireYamllint(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("yamllint"); err != nil {
		t.Skip("yamllint not installed")
	}
}

func TestYAMLChecker_Name(t *testing.T) {
	assert.Equal(t, "yaml", checks.NewYAMLChecker().Name())
}

func TestYAMLChecker_Check(t *testing.T) {
	t.Run("warning-level problems do not fail the check", func(t *testing.T) {
		requireYamllint(t)

		path := writeTemp(t, "long.yaml", longLineYAML)

		result, err := checks.NewYAMLChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.False(t, result.HasViolations(), "yamllint exits 0 on warnings; the check must agree")
	})

	t.Run("missing newline at end of file is a violation", func(t *testing.T) {
		requireYamllint(t)

		path := writeTemp(t, "nonewline.yaml", "---\nkey: value")

		result, err := checks.NewYAMLChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.True(t, result.HasViolations(), "error-level problems must still fail")
	})

	t.Run("syntax error is a violation", func(t *testing.T) {
		requireYamllint(t)

		path := writeTemp(t, "bad.yaml", "---\nkey: [unclosed\n")

		result, err := checks.NewYAMLChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.True(t, result.HasViolations())
	})

	t.Run("violation carries the file path and line number", func(t *testing.T) {
		requireYamllint(t)

		path := writeTemp(t, "bad.yaml", "---\nkey: [unclosed\n")

		result, err := checks.NewYAMLChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, path, result.Violations[0].File)
		assert.Positive(t, result.Violations[0].Line)
		assert.NotContains(t, result.Violations[0].Message, path, "the path is already in File")
	})

	t.Run("yamllint header lines are not reported as violations", func(t *testing.T) {
		requireYamllint(t)

		clean := writeTemp(t, "clean.yaml", "---\nkey: value\n")

		result, err := checks.NewYAMLChecker().Check(t.Context(), []string{clean})
		require.NoError(t, err)
		assert.Empty(t, result.Violations, "a clean file must produce no output at all")
	})

	t.Run("skips non-YAML files", func(t *testing.T) {
		path := writeTemp(t, "config.toml", "key = 'val'\n")

		result, err := checks.NewYAMLChecker().Check(t.Context(), []string{path})
		require.NoError(t, err)
		assert.False(t, result.HasViolations(), "non-YAML files must be skipped")
	})
}
