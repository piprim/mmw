package scaffold_test

import (
	"io/fs"
	"testing"

	"github.com/piprim/goplt"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedFS(t *testing.T) {
	fsys := scaffold.EmbeddedFS()

	t.Run("top-level entries are modules, contracts and goplt.toml", func(t *testing.T) {
		entries, err := fs.ReadDir(fsys, ".")
		require.NoError(t, err)

		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}

		assert.ElementsMatch(t, []string{"modules", "contracts", "goplt.toml"}, names)
	})

	t.Run("manifest declares name, entity, database, create-table and grpc only", func(t *testing.T) {
		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)

		names := make([]string, len(m.Variables))
		for i, v := range m.Variables {
			names[i] = v.Name
		}

		assert.ElementsMatch(t, []string{"Name", "Entity", "Database", "CreateTable", "Grpc"}, names)
		assert.Equal(t, [2]string{"[[", "]]"}, m.Delimiters)
	})
}
