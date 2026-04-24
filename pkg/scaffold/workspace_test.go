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
	dir := t.TempDir()

	// Seed a minimal go.work
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
	// Original entry preserved
	assert.Contains(t, string(content), "./modules/todo")
}

func TestUpdateGoWork_WithReplaceBlock(t *testing.T) {
	dir := t.TempDir()
	// replace () appears before use () — the old strings.Replace(…, 1) would
	// have inserted into the replace block instead of the use block.
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
	// replace block must be untouched
	assert.Contains(t, string(content), "github.com/foo/bar => ./local-bar")
}

func TestUpdateGoWork_Idempotent(t *testing.T) {
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
	// Should not appear twice
	count := strings.Count(string(content), "./modules/payment")
	assert.Equal(t, 1, count)
}

func TestUpdateMiseToml(t *testing.T) {
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
}
