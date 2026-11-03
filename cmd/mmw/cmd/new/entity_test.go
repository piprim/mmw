package new

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piprim/goplt"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveModule(t *testing.T) {
	t.Run("flag wins over the positional argument", func(t *testing.T) {
		got, err := resolveModule("/r", "a", "b")
		require.NoError(t, err)
		assert.Equal(t, "b", got)
	})

	t.Run("accepts a path to the module directory", func(t *testing.T) {
		got, err := resolveModule("/r", "modules/facturation/", "")
		require.NoError(t, err)
		assert.Equal(t, "facturation", got)
	})

	t.Run("empty means ask", func(t *testing.T) {
		got, err := resolveModule("/r", "", "")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestListModules(t *testing.T) {
	t.Run("lists directories with a <name>.go, sorted", func(t *testing.T) {
		root := t.TempDir()
		for _, n := range []string{"b", "a"} {
			require.NoError(t, os.MkdirAll(filepath.Join(root, "modules", n), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(root, "modules", n, n+".go"), nil, 0o600))
		}
		require.NoError(t, os.MkdirAll(filepath.Join(root, "modules", "junk"), 0o755))

		got, err := listModules(root)
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, got)
	})
}

func TestEntityPromptOrder(t *testing.T) {
	t.Run("module, entity, shape, create-table", func(t *testing.T) {
		m, err := goplt.LoadManifest(scaffold.SliceFS())
		require.NoError(t, err)
		orderVars(m)
		var got []string
		for _, v := range m.Variables {
			got = append(got, v.Name)
		}
		assert.Equal(t, []string{"Module", "Entity", "Shape", "CreateTable"}, got)
	})
}

func TestNarrowShapes(t *testing.T) {
	t.Run("a module without a database offers gateway only", func(t *testing.T) {
		m, err := goplt.LoadManifest(scaffold.SliceFS())
		require.NoError(t, err)
		narrowShapes(m, false)
		for _, v := range m.Variables {
			if v.Name == "Shape" {
				assert.Equal(t, []string{"gateway"}, v.Value)
			}
		}
	})
}
