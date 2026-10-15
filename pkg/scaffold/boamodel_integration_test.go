//go:build integration

package scaffold_test

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pfdbmigrator "github.com/piprim/mmw/pkg/platform/db/migrator"
)

const boamodelRel = "internal/adapters/outbound/persistence/postgres/dao/boamodel"

// TestBoamodelMatchesBungen proves the shipped dao/boamodel/{models,columns,q}.go
// are what bungen emits for the rendered migrations and templates.
// Set SCAFFOLD_UPDATE_BOAMODEL=1 to overwrite the three templates from the run.
func TestBoamodelMatchesBungen(t *testing.T) {
	bungen, err := exec.LookPath("bungen")
	if err != nil {
		t.Skip("bungen not on PATH")
	}

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
			cmd := exec.CommandContext(ctx, bungen, "model", "-c", dsn, "-t", "facturation.event,facturation.facture", "-f",
				"--tag", "boa", "--presence", "-o", got, "--template", filepath.Join(mod, boamodelRel, name+".go.tmpl"))
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

// templatesRoot is pkg/scaffold/_templates on disk (the test runs in pkg/scaffold).
func templatesRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)

	return filepath.Join(wd, "_templates")
}
