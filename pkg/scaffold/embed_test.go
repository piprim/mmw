package scaffold_test

import (
	"io/fs"
	"testing"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedFS(t *testing.T) {
	t.Run("top-level entries include modules, contracts, and template.toml", func(t *testing.T) {
		fsys := scaffold.EmbeddedFS()

		entries, err := fs.ReadDir(fsys, ".")
		require.NoError(t, err)

		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}

		assert.Contains(t, names, "modules")
		assert.Contains(t, names, "contracts")
		assert.Contains(t, names, "template.toml")
	})

	t.Run("template.toml is present at FS root", func(t *testing.T) {
		fsys := scaffold.EmbeddedFS()
		_, err := fs.Stat(fsys, "template.toml")
		assert.NoError(t, err)
	})
}
