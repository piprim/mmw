package scaffold_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTemp writes content to a fresh file and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "f.go")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(b)
}

func TestInsertBefore(t *testing.T) {
	const src = "type M struct {\n\tx int\n\t// mmw:slice-field\n}\n"

	t.Run("inserts before the marker keeping its indentation", func(t *testing.T) {
		path := writeTemp(t, src)
		ok, err := scaffold.InsertBefore(path, scaffold.MarkerField, "y int")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "type M struct {\n\tx int\n\ty int\n\t// mmw:slice-field\n}\n", readFile(t, path))
	})

	t.Run("keeps the marker for the next slice", func(t *testing.T) {
		path := writeTemp(t, src)
		_, err := scaffold.InsertBefore(path, scaffold.MarkerField, "y int")
		require.NoError(t, err)
		_, err = scaffold.InsertBefore(path, scaffold.MarkerField, "z int")
		require.NoError(t, err)
		assert.Equal(t, "type M struct {\n\tx int\n\ty int\n\tz int\n\t// mmw:slice-field\n}\n", readFile(t, path))
	})

	t.Run("returns false when the marker is absent", func(t *testing.T) {
		path := writeTemp(t, src)
		ok, err := scaffold.InsertBefore(path, scaffold.MarkerWire, "y int")
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, src, readFile(t, path))
	})

	t.Run("does not insert a snippet that is already there", func(t *testing.T) {
		path := writeTemp(t, src)
		_, err := scaffold.InsertBefore(path, scaffold.MarkerField, "y int")
		require.NoError(t, err)
		once := readFile(t, path)
		ok, err := scaffold.InsertBefore(path, scaffold.MarkerField, "y int")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, once, readFile(t, path))
	})

	t.Run("inserts a multi-line snippet", func(t *testing.T) {
		path := writeTemp(t, "func f() error {\n\t// mmw:slice-error\n\treturn nil\n}\n")
		ok, err := scaffold.InsertBefore(path, scaffold.MarkerError, "if d := g(); d != nil {\n\treturn d\n}")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t,
			"func f() error {\n\tif d := g(); d != nil {\n\t\treturn d\n\t}\n\t// mmw:slice-error\n\treturn nil\n}\n",
			readFile(t, path))
	})

	t.Run("fails without writing when the result is not valid Go", func(t *testing.T) {
		const goSrc = "package p\n\nfunc f() error {\n\t// mmw:slice-error\n\treturn nil\n}\n"
		path := writeTemp(t, goSrc)
		_, err := scaffold.InsertBefore(path, scaffold.MarkerError, "if d := ; {")
		require.Error(t, err)
		assert.Contains(t, err.Error(), path)
		assert.Equal(t, goSrc, readFile(t, path))
	})

	t.Run("fails on a missing file", func(t *testing.T) {
		_, err := scaffold.InsertBefore(filepath.Join(t.TempDir(), "nope.go"), scaffold.MarkerField, "y int")
		assert.Error(t, err)
	})
}

func TestAppendBungenTable(t *testing.T) {
	const mise = "run = \"\"\"\n  bungen model -c 'x' -t 'f.event' -f\n\"\"\"\n"

	t.Run("appends to the -t list", func(t *testing.T) {
		path := writeTemp(t, mise)
		ok, err := scaffold.AppendBungenTable(path, "f.facture")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Contains(t, readFile(t, path), "-t 'f.event,f.facture'")
	})

	t.Run("is idempotent", func(t *testing.T) {
		path := writeTemp(t, mise)
		_, err := scaffold.AppendBungenTable(path, "f.facture")
		require.NoError(t, err)
		ok, err := scaffold.AppendBungenTable(path, "f.facture")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Contains(t, readFile(t, path), "-t 'f.event,f.facture' -f")
	})

	t.Run("returns false without a -t list", func(t *testing.T) {
		path := writeTemp(t, "[tasks.lint]\nrun = \"golangci-lint run\"\n")
		ok, err := scaffold.AppendBungenTable(path, "f.facture")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
