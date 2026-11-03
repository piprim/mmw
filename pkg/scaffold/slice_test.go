package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sliceVars returns the fixture variables with a different entity and shape.
func sliceVars(entity, shape string) map[string]any {
	v := fixtureVars()
	v["Entity"] = entity
	v["Shape"] = shape
	delete(v, "TimestampCreate")
	delete(v, "TimestampSeed")

	return v
}

// newModuleRoot renders the facturation module and its first slice (facture)
// into a temp repo root with a stub contracts/go.mod.
func newModuleRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "contracts", "go.mod"), "module ovya.fr/rcv/contracts\n")
	vars := fixtureVars()
	require.NoError(t, scaffold.Render(scaffold.ModuleFS(), root, vars))
	_, err := scaffold.AddSlice(scaffold.SliceFS(), root, vars)
	require.NoError(t, err)

	return root
}

func TestDetectModule(t *testing.T) {
	root := newModuleRoot(t)

	t.Run("reads database, grpc and module path from the tree", func(t *testing.T) {
		got, err := scaffold.DetectModule(root, "facturation")
		require.NoError(t, err)
		assert.Equal(t, scaffold.ModuleTraits{
			Name: "facturation", ModulePath: "ovya.fr/rcv/facturation", Database: true, Grpc: true,
		}, got)
	})

	t.Run("a module without cmd/<name>", func(t *testing.T) {
		require.NoError(t, os.RemoveAll(filepath.Join(root, "modules", "facturation", "cmd", "facturation")))
		got, err := scaffold.DetectModule(root, "facturation")
		require.NoError(t, err)
		t.Run("is in-proc only", func(t *testing.T) { assert.False(t, got.Grpc) })
		t.Run("keeps its database", func(t *testing.T) { assert.True(t, got.Database) })
	})

	t.Run("fails when modules/<name>/<name>.go is missing", func(t *testing.T) {
		_, err := scaffold.DetectModule(root, "absent")
		assert.Error(t, err)
	})

	t.Run("Apply sets the module traits in vars", func(t *testing.T) {
		vars := map[string]any{}
		scaffold.ModuleTraits{Name: "x", ModulePath: "a/x", Database: true}.Apply(vars)
		assert.Equal(t, map[string]any{"Name": "x", "ModulePath": "a/x", "Database": true, "Grpc": false}, vars)
	})
}

func TestMigrationStamps(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

	t.Run("uses the clock when the directory is empty", func(t *testing.T) {
		create, seed := scaffold.MigrationStamps(t.TempDir(), now)
		assert.Equal(t, "20261103100000", create)
		assert.Equal(t, "20261103100001", seed)
	})

	t.Run("uses the clock when the directory is missing", func(t *testing.T) {
		create, _ := scaffold.MigrationStamps(filepath.Join(t.TempDir(), "nope"), now)
		assert.Equal(t, "20261103100000", create)
	})

	t.Run("stamps after the highest existing migration", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "20261103100005_x.sql"), "")
		create, seed := scaffold.MigrationStamps(dir, now)
		assert.Equal(t, "20261103100006", create)
		assert.Equal(t, "20261103100007", seed)
	})

	t.Run("ignores files without a 14-digit prefix", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "99999999999999999_long.sql"), "")
		writeFile(t, filepath.Join(dir, "readme.md"), "")
		writeFile(t, filepath.Join(dir, "2099_short.sql"), "")
		create, _ := scaffold.MigrationStamps(dir, now)
		assert.Equal(t, "20261103100000", create)
	})
}

func TestValidateSlice(t *testing.T) {
	root := newModuleRoot(t)
	vars := func(entity, shape string, db bool) map[string]any {
		return map[string]any{"Name": "facturation", "Entity": entity, "Shape": shape, "Database": db}
	}

	t.Run("rejects an entity whose domain package exists", func(t *testing.T) {
		assert.ErrorContains(t, scaffold.ValidateSlice(root, vars("facture", "aggregate", true)), "already exists")
	})

	t.Run("rejects an entity whose proto exists", func(t *testing.T) {
		writeFile(t, filepath.Join(root, "contracts", "proto", "facturation", "v1", "devis.proto"), "")
		assert.ErrorContains(t, scaffold.ValidateSlice(root, vars("devis", "aggregate", true)), "already exists")
	})

	t.Run("rejects a shape the module cannot host", func(t *testing.T) {
		assert.ErrorContains(t, scaffold.ValidateSlice(root, vars("avoir", "read-model", false)), "shape")
	})

	t.Run("rejects a reserved entity", func(t *testing.T) {
		assert.ErrorContains(t, scaffold.ValidateSlice(root, vars("domain", "aggregate", true)), "reserved")
	})

	for _, e := range []string{"helpers", "platform", "error", "any"} {
		t.Run("rejects reserved entity "+e, func(t *testing.T) {
			assert.ErrorContains(t, scaffold.ValidateSlice(root, vars(e, "aggregate", true)), "reserved")
		})
	}

	t.Run("accepts a second entity", func(t *testing.T) {
		assert.NoError(t, scaffold.ValidateSlice(root, vars("avoir", "read-model", true)))
	})
}

func TestAddSlice(t *testing.T) {
	root := newModuleRoot(t)
	mod := "modules/facturation"

	avoir := sliceVars("avoir", "read-model")
	rep, err := scaffold.AddSlice(scaffold.SliceFS(), root, avoir)
	require.NoError(t, err)
	relance := sliceVars("relance", "gateway")
	rep2, err := scaffold.AddSlice(scaffold.SliceFS(), root, relance)
	require.NoError(t, err)

	t.Run("renders the slice files", func(t *testing.T) {
		assertFileExists(t, root, mod+"/internal/domain/avoir")
		assertFileExists(t, root, mod+"/internal/domain/relance")
		assertFileExists(t, root, mod+"/avoir_wiring.go")
		assertFileExists(t, root, mod+"/relance_wiring.go")
	})

	for _, want := range []string{"m.facture =", "m.avoir =", "m.relance =", "// mmw:slice-wire"} {
		t.Run("facturation.go has "+want, func(t *testing.T) {
			assertFileContains(t, root, mod+"/facturation.go", want)
		})
	}

	t.Run("registers the aggregate topic", func(t *testing.T) {
		assertFileContains(t, root, mod+"/internal/adapters/outbound/events/topics.go", "factureEventType")
	})
	t.Run("registers the gateway topic", func(t *testing.T) {
		assertFileContains(t, root, mod+"/internal/adapters/outbound/events/topics.go", "relanceEventType")
	})
	t.Run("registers no topic for a read model", func(t *testing.T) {
		assertFileNotContains(t, root, mod+"/internal/adapters/outbound/events/topics.go", "avoirEventType")
	})

	t.Run("lists aggregate and read-model tables", func(t *testing.T) {
		assertFileContains(t, root, mod+"/mise.toml", "-t 'facturation.event,facturation.facture,facturation.avoir'")
	})
	t.Run("does not list the gateway table", func(t *testing.T) {
		assertFileNotContains(t, root, mod+"/mise.toml", "facturation.relance")
	})

	t.Run("reports nothing unwired on a generated module", func(t *testing.T) {
		assert.Empty(t, rep.Unwired)
		assert.Empty(t, rep2.Unwired)
	})
	t.Run("reports no unlisted table on a generated module", func(t *testing.T) {
		assert.Empty(t, rep.TableUnlisted)
	})

	t.Run("rendered tree stays gofmt-clean", func(t *testing.T) {
		assertGofmt(t, root)
	})

	t.Run("two slices in one second get distinct stamps", func(t *testing.T) {
		assert.NotEqual(t, avoir["TimestampCreate"], relance["TimestampCreate"])
	})
	t.Run("a later slice stamps after an earlier one", func(t *testing.T) {
		assert.Greater(t, avoir["TimestampSeed"], avoir["TimestampCreate"])
		assert.Greater(t, avoir["TimestampCreate"], "20261013120000")
	})

	t.Run("reports the snippets when markers are missing", func(t *testing.T) {
		path := filepath.Join(root, mod, "facturation.go")
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var kept []string
		for l := range strings.SplitSeq(string(data), "\n") {
			if !strings.Contains(l, "mmw:slice-") {
				kept = append(kept, l)
			}
		}
		require.NoError(t, os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o600))

		rep, err := scaffold.AddSlice(scaffold.SliceFS(), root, sliceVars("devis", "aggregate"))
		require.NoError(t, err)
		require.Len(t, rep.Unwired, 3)
		assert.Equal(t, "facturation.go", rep.Unwired[0].Path)
	})
	t.Run("reports the table when the -t list is missing", func(t *testing.T) {
		path := filepath.Join(root, mod, "mise.toml")
		require.NoError(t, os.WriteFile(path, []byte("[tasks]\n"), 0o600))
		rep, err := scaffold.AddSlice(scaffold.SliceFS(), root, sliceVars("bon", "aggregate"))
		require.NoError(t, err)
		assert.Equal(t, "facturation.bon", rep.TableUnlisted)
	})
}
