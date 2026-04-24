package os_test

import (
	"testing"

	pfos "github.com/piprim/mmw/pkg/platform/os"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvMap(t *testing.T) {
	t.Run("returns non-nil map", func(t *testing.T) {
		assert.NotNil(t, pfos.EnvMap())
	})

	t.Run("contains PATH which is always set", func(t *testing.T) {
		_, ok := pfos.EnvMap()["PATH"]
		assert.True(t, ok)
	})

	t.Run("parses value correctly", func(t *testing.T) {
		t.Setenv("_TEST_ENVMAP_KEY", "hello_world")

		m := pfos.EnvMap()
		require.Contains(t, m, "_TEST_ENVMAP_KEY")
		assert.Equal(t, "hello_world", m["_TEST_ENVMAP_KEY"])
	})

	t.Run("includes keys with empty value", func(t *testing.T) {
		t.Setenv("_TEST_ENVMAP_EMPTY", "")

		m := pfos.EnvMap()
		val, ok := m["_TEST_ENVMAP_EMPTY"]
		assert.True(t, ok)
		assert.Equal(t, "", val)
	})

	t.Run("uses first equals sign as delimiter when value contains equals", func(t *testing.T) {
		t.Setenv("_TEST_ENVMAP_EQ", "a=b=c")

		m := pfos.EnvMap()
		require.Contains(t, m, "_TEST_ENVMAP_EQ")
		assert.Equal(t, "a=b=c", m["_TEST_ENVMAP_EQ"])
	})
}
