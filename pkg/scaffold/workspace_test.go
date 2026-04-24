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
	t.Run("inserts new module path into use block", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.work"), []byte(`go 1.26.1

use (
	.
	./modules/todo
)
`), 0600))

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
`), 0600))

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
`), 0600))

		require.NoError(t, scaffold.UpdateGoWork(dir, "payment"))

		content, _ := os.ReadFile(filepath.Join(dir, "go.work"))
		assert.Equal(t, 1, strings.Count(string(content), "./modules/payment"))
	})
}

func TestUpdateMiseToml(t *testing.T) {
	t.Run("appends test and test:contract tasks for new module", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "mise.toml"), []byte(`[tasks."todo:test"]
run = "cd modules/todo && mise run test"
`), 0600))

		require.NoError(t, scaffold.UpdateMiseToml(dir, "payment"))

		content, _ := os.ReadFile(filepath.Join(dir, "mise.toml"))
		assert.Contains(t, string(content), `[tasks."payment:test"]`)
		assert.Contains(t, string(content), `cd modules/payment && mise run test`)
		assert.Contains(t, string(content), `[tasks."payment:test:contract"]`)
		assert.Contains(t, string(content), `cd modules/payment && mise run test:contract`)
	})
}
