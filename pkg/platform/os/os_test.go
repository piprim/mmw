package os_test

import (
	"testing"

	pfos "github.com/piprim/mmw/pkg/platform/os"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvMap_NotNil(t *testing.T) {
	m := pfos.EnvMap()
	assert.NotNil(t, m)
}

func TestEnvMap_ContainsKnownVar(t *testing.T) {
	// PATH is always set in any Unix/Linux/macOS environment.
	m := pfos.EnvMap()
	_, ok := m["PATH"]
	assert.True(t, ok, "PATH should be present in EnvMap")
}

func TestEnvMap_ParsesValue(t *testing.T) {
	t.Setenv("_TEST_ENVMAP_KEY", "hello_world")

	m := pfos.EnvMap()
	require.Contains(t, m, "_TEST_ENVMAP_KEY")
	assert.Equal(t, "hello_world", m["_TEST_ENVMAP_KEY"])
}

func TestEnvMap_EmptyValue(t *testing.T) {
	t.Setenv("_TEST_ENVMAP_EMPTY", "")

	m := pfos.EnvMap()
	val, ok := m["_TEST_ENVMAP_EMPTY"]
	assert.True(t, ok)
	assert.Equal(t, "", val)
}

func TestEnvMap_ValueContainsEquals(t *testing.T) {
	// Only the first '=' is used as the delimiter; value may itself contain '='.
	t.Setenv("_TEST_ENVMAP_EQ", "a=b=c")

	m := pfos.EnvMap()
	require.Contains(t, m, "_TEST_ENVMAP_EQ")
	assert.Equal(t, "a=b=c", m["_TEST_ENVMAP_EQ"])
}
