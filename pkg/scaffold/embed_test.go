package scaffold_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/piprim/goplt"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func varNames(m *goplt.Manifest) []string {
	names := make([]string, len(m.Variables))
	for i, v := range m.Variables {
		names[i] = v.Name
	}

	return names
}

func TestModuleFS(t *testing.T) {
	fsys := scaffold.ModuleFS()

	t.Run("top-level entries are modules, contracts and goplt.toml", func(t *testing.T) {
		entries, err := fs.ReadDir(fsys, ".")
		require.NoError(t, err)

		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}

		assert.ElementsMatch(t, []string{"modules", "contracts", "goplt.toml"}, names)
	})

	t.Run("manifest declares name, database and grpc", func(t *testing.T) {
		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)

		assert.ElementsMatch(t, []string{"Name", "Database", "Grpc"}, varNames(m))
		assert.Equal(t, [2]string{"[[", "]]"}, m.Delimiters)
	})

	t.Run("no path contains [[.Entity]] except dao/boamodel", func(t *testing.T) {
		var offenders []string
		require.NoError(t, fs.WalkDir(fsys, ".", func(p string, _ fs.DirEntry, err error) error {
			if err == nil && strings.Contains(p, "[[.Entity]]") && !strings.Contains(p, "dao/boamodel/") {
				offenders = append(offenders, p)
			}

			return err
		}))
		assert.Empty(t, offenders)
	})
}

func TestSliceFS(t *testing.T) {
	fsys := scaffold.SliceFS()

	t.Run("manifest declares module, entity, shape and create-table", func(t *testing.T) {
		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)

		assert.ElementsMatch(t, []string{"Module", "Entity", "Shape", "CreateTable"}, varNames(m))
		assert.Equal(t, [2]string{"[[", "]]"}, m.Delimiters)
	})

	t.Run("every file path or content names the entity", func(t *testing.T) {
		var offenders []string
		require.NoError(t, fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || p == "goplt.toml" || strings.Contains(p, "[[.Entity]]") {
				return err
			}
			data, rerr := fs.ReadFile(fsys, p)
			if rerr != nil {
				return rerr
			}
			if !strings.Contains(string(data), "[[.Entity") {
				offenders = append(offenders, p)
			}

			return nil
		}))
		assert.Empty(t, offenders)
	})
}

func TestMergeManifests(t *testing.T) {
	mod, err := goplt.LoadManifest(scaffold.ModuleFS())
	require.NoError(t, err)
	sl, err := goplt.LoadManifest(scaffold.SliceFS())
	require.NoError(t, err)
	merged := scaffold.MergeManifests(mod, sl)

	t.Run("module variables first, then slice variables without Module", func(t *testing.T) {
		got := varNames(merged)
		// goplt sorts variables on load, so the slice part is CreateTable, Entity, Shape.
		assert.Equal(t, append(varNames(mod), "CreateTable", "Entity", "Shape"), got)
	})

	t.Run("delimiters are [[ ]]", func(t *testing.T) {
		assert.Equal(t, [2]string{"[[", "]]"}, merged.Delimiters)
	})
}
