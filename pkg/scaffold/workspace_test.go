package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateGoWork(t *testing.T) {
	t.Run("adds notification when notifications is already present", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.26.1\n\nuse (\n\t.\n\t./modules/notifications\n)\n"), 0o600))

		require.NoError(t, scaffold.UpdateGoWork(dir, "notification"))

		content, err := os.ReadFile(filepath.Join(dir, "go.work"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "\t./modules/notification\n")
	})

	t.Run("inserts new module path into use block", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.work"), []byte(`go 1.26.1

use (
	.
	./modules/todo
)
`), 0o600))

		require.NoError(t, scaffold.UpdateGoWork(dir, "payment"))

		content, err := os.ReadFile(filepath.Join(dir, "go.work"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "./modules/payment")
		assert.Contains(t, string(content), "./modules/todo")
	})

	t.Run("inserts into use block not replace block when both exist", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.work"), []byte(`go 1.26.1

replace (
	github.com/foo/bar => ./local-bar
)

use (
	.
	./modules/todo
)
`), 0o600))

		require.NoError(t, scaffold.UpdateGoWork(dir, "payment"))

		content, err := os.ReadFile(filepath.Join(dir, "go.work"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "./modules/payment")
		assert.Contains(t, string(content), "github.com/foo/bar => ./local-bar")
	})

	t.Run("is idempotent when module already present", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.work"), []byte(`go 1.26.1

use (
	.
	./modules/todo
	./modules/payment
)
`), 0o600))

		require.NoError(t, scaffold.UpdateGoWork(dir, "payment"))

		content, _ := os.ReadFile(filepath.Join(dir, "go.work"))
		assert.Equal(t, 1, strings.Count(string(content), "./modules/payment"))
	})
}

func TestUpdateMiseToml(t *testing.T) {
	write := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "mise.toml"), []byte("[tasks.x]\nrun = \"true\"\n"), 0o644))

		return root
	}
	read := func(t *testing.T, root string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, "mise.toml"))
		require.NoError(t, err)

		return string(b)
	}

	t.Run("database and grpc add test, integration, contract and run tasks", func(t *testing.T) {
		root := write(t)
		require.NoError(t, scaffold.UpdateMiseToml(root, "facturation", true, true))
		got := read(t, root)
		for _, task := range []string{`[tasks."facturation:test"]`, `[tasks."facturation:test:integration"]`, `[tasks."facturation:test:contract"]`, `[tasks."facturation:run"]`} {
			assert.Contains(t, got, task)
		}
	})

	t.Run("gateway in-proc module gets only test and contract tasks", func(t *testing.T) {
		root := write(t)
		require.NoError(t, scaffold.UpdateMiseToml(root, "messagerie", false, false))
		got := read(t, root)
		assert.Contains(t, got, `[tasks."messagerie:test"]`)
		assert.Contains(t, got, `[tasks."messagerie:test:contract"]`)
		assert.NotContains(t, got, "test:integration")
		assert.NotContains(t, got, `"messagerie:run"`)
	})

	t.Run("is idempotent", func(t *testing.T) {
		root := write(t)
		require.NoError(t, scaffold.UpdateMiseToml(root, "facturation", true, true))
		require.NoError(t, scaffold.UpdateMiseToml(root, "facturation", true, true))
		assert.Equal(t, 1, strings.Count(read(t, root), `[tasks."facturation:test"]`))
	})
}
