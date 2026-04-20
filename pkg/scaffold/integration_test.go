//go:build integration

package scaffold_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piprim/goplt"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerate_FullOptionOutput generates a module into a temp directory
// that mirrors the real repo structure and verifies the files are generated correctly.
func TestGenerate_FullOptionOutput(t *testing.T) {
	repoRoot := findRepoRoot(t)

	dir := t.TempDir()

	// Seed go.work
	require.NoError(t, copyFile(
		filepath.Join(repoRoot, "go.work"),
		filepath.Join(dir, "go.work"),
	))

	fsys := scaffold.EmbeddedFS()

	m, err := goplt.LoadManifest(fsys)
	require.NoError(t, err)

	vars := map[string]any{
		"Name":          "demomod",
		"OrgPrefix":     "github.com/pivaldi",
		"ContractsPath": "github.com/pivaldi/mmw-contracts",
		"PlatformPath":  "github.com/piprim/mmw",
		"WithModule":    true,
		"WithConnect":   true,
		"WithContract":  true,
		"WithDatabase":  true,
		"License":       "MIT",
	}

	require.NoError(t, goplt.NewGenerator().Generate(fsys, m, dir, vars))
	require.NoError(t, scaffold.UpdateGoWork(dir, "demomod"))

	// Verify go.work now contains demomod
	goWorkContent, err := os.ReadFile(filepath.Join(dir, "go.work"))
	require.NoError(t, err)
	assert.Contains(t, string(goWorkContent), "./modules/demomod")

	// Verify key files generated
	assertFileExists(t, dir, "modules/demomod/go.mod")
	assertFileContains(t, dir, "modules/demomod/go.mod", "module github.com/pivaldi/demomod")
	assertFileExists(t, dir, "modules/demomod/demomodmod.go")
	assertFileExists(t, dir, "contracts/go/application/demomod/api.go")
	assertFileExists(t, dir, "contracts/proto/demomod/v1/demomod.proto")
	assertFileExists(t, dir, "modules/demomod/internal/infra/persistence/migrations/migrations.go")

	t.Logf("Generated module at: %s/modules/demomod", dir)
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (go.work)")
		}
		dir = parent
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
