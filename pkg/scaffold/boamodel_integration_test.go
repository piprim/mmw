//go:build integration

package scaffold_test

import (
	"bytes"
	"context"
	"database/sql"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pfdbmigrator "github.com/piprim/mmw/pkg/platform/db/migrator"
	"github.com/piprim/mmw/pkg/scaffold"
)

const boamodelRel = "internal/adapters/outbound/persistence/postgres/dao/boamodel"

// TestBoamodelMatchesBungen proves the shipped dao/boamodel/{models,columns,q}.go
// are what bungen emits for the rendered migrations and templates.
// Set SCAFFOLD_UPDATE_BOAMODEL=1 to overwrite the three templates from the run.
func TestBoamodelMatchesBungen(t *testing.T) {
	bungen := resolveBungen(t)

	dir := renderFixture(t)
	mod := filepath.Join(dir, "modules", "facturation")
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"), postgres.WithUsername("testuser"), postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()
	scripts := filepath.Join(mod, "internal/infra/persistence/migrations/scripts")
	m, err := pfdbmigrator.New(db, os.DirFS(filepath.Dir(scripts)), "scripts", "facturation")
	require.NoError(t, err)
	_, err = m.Up(ctx)
	require.NoError(t, err)

	out := t.TempDir()
	for _, name := range []string{"models", "columns", "q"} {
		t.Run(name+".go matches bungen", func(t *testing.T) {
			got := filepath.Join(out, name+".go")
			cmd := exec.CommandContext(ctx, bungen[0], append(bungen[1:], "model", "-c", dsn, "-t", "facturation.event,facturation.facture", "-f",
				"--tag", "boa", "--presence", "-p", "boamodel", "-o", got, "--template", filepath.Join(mod, boamodelRel, name+".go.tmpl"))...)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, string(output))

			want, err := os.ReadFile(filepath.Join(mod, boamodelRel, name+".go"))
			require.NoError(t, err)
			actual, err := os.ReadFile(got)
			require.NoError(t, err)

			if os.Getenv("SCAFFOLD_UPDATE_BOAMODEL") == "1" {
				tmpl := filepath.Join(templatesRoot(t), "modules", "[[.Name]]", boamodelRel, name+".go")
				require.NoError(t, os.WriteFile(tmpl, []byte(templatize(string(actual))), 0o644))
				t.Logf("updated %s", tmpl)

				return
			}

			require.Equal(t, string(want), string(actual), "run with SCAFFOLD_UPDATE_BOAMODEL=1 to refresh the template")
		})
	}

	t.Run("a second aggregate's table is in the -t list and bungen emits its model", func(t *testing.T) {
		vars := fixtureVars()
		vars["Entity"] = "avoir"
		restoreTree(t, dir) // renderFixture's tree is shared with TestRender
		_, err = scaffold.AddSlice(scaffold.SliceFS(), dir, vars)
		require.NoError(t, err)

		_, err = m.Up(ctx)
		require.NoError(t, err)

		mise, err := os.ReadFile(filepath.Join(mod, "mise.toml"))
		require.NoError(t, err)
		match := regexp.MustCompile(`-t '([^']*)'`).FindSubmatch(mise)
		require.NotNil(t, match, "no -t list in mise.toml")

		got := filepath.Join(t.TempDir(), "models.go")
		output, err := exec.CommandContext(ctx, bungen[0], append(bungen[1:], "model", "-c", dsn, "-t", string(match[1]), "-f",
			"--tag", "boa", "--presence", "-p", "boamodel", "-o", got, "--template", filepath.Join(mod, boamodelRel, "models.go.tmpl"))...).CombinedOutput()
		require.NoError(t, err, string(output))

		actual, err := os.ReadFile(got)
		require.NoError(t, err)
		require.Contains(t, string(actual), "type FacturationAvoir struct")
	})
}

// resolveBungen returns the bungen command: $BUNGEN (space-separated), else the
// repository's fork (tools/bungen, which has --template/--presence) built once.
// A bungen on PATH is deliberately not used: upstream ignores those flags.
func resolveBungen(t *testing.T) []string {
	t.Helper()
	if env := os.Getenv("BUNGEN"); env != "" {
		return strings.Fields(env)
	}
	wd, err := os.Getwd()
	require.NoError(t, err)
	// the test runs in <mmw>/pkg/scaffold; the fork is <cpro2>/tools/bungen
	src := filepath.Join(wd, "..", "..", "..", "tools", "bungen")
	if _, err := os.Stat(filepath.Join(src, "main.go")); err != nil {
		t.Skip("bungen fork not found: set BUNGEN or check out mmw inside cpro2")
	}
	bin := filepath.Join(t.TempDir(), "bungen")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = src
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	return []string{bin}
}

// restoreTree snapshots every file under root and, at cleanup, removes files
// added since and rewrites files changed since.
func restoreTree(t *testing.T, root string) {
	t.Helper()
	before := map[string][]byte{}
	require.NoError(t, filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		before[p] = b

		return err
	}))
	t.Cleanup(func() {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if _, ok := before[p]; !ok {
				return os.Remove(p)
			}

			return nil
		})
		for p, b := range before {
			if cur, err := os.ReadFile(p); err != nil || !bytes.Equal(cur, b) {
				_ = os.WriteFile(p, b, 0o644)
			}
		}
	})
}

// templatize turns the fixture's rendered names back into template actions.
func templatize(s string) string {
	r := strings.NewReplacer(
		"FacturationFacture", "[[.Name | pascal]][[.Entity | pascal]]",
		"FacturationEvent", "[[.Name | pascal]]Event",
		"facturation.facture", "[[.Name]].[[.Entity]]",
		"facturation.event", "[[.Name]].event",
	)

	return r.Replace(s)
}

// templatesRoot is pkg/scaffold/_templates/module on disk (the test runs in pkg/scaffold).
func templatesRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)

	return filepath.Join(wd, "_templates", "module")
}
