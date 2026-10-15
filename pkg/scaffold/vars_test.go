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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestDetect(t *testing.T) {
	t.Run("reads the org prefix and contracts path from contracts/go.mod", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "contracts", "go.mod"), "module ovya.fr/rcv/contracts\n\ngo 1.27.1\n")

		d, err := scaffold.Detect(root)
		require.NoError(t, err)
		assert.Equal(t, "ovya.fr/rcv", d.OrgPrefix)
		assert.Equal(t, "ovya.fr/rcv/contracts", d.ContractsPath)
		assert.Equal(t, "github.com/piprim/mmw", d.PlatformPath)
	})

	t.Run("fails without contracts/go.mod", func(t *testing.T) {
		_, err := scaffold.Detect(t.TempDir())
		assert.ErrorContains(t, err, "contracts/go.mod")
	})

	t.Run("Apply sets the three detected keys", func(t *testing.T) {
		vars := map[string]any{"Name": "x"}
		scaffold.Detected{OrgPrefix: "a", ContractsPath: "a/c", PlatformPath: "p"}.Apply(vars)
		assert.Equal(t, map[string]any{"Name": "x", "OrgPrefix": "a", "ContractsPath": "a/c", "PlatformPath": "p"}, vars)
	})
}

func loadManifest(t *testing.T) *goplt.Manifest {
	t.Helper()
	m, err := goplt.LoadManifest(scaffold.EmbeddedFS())
	require.NoError(t, err)

	return m
}

func TestParseVarFlags(t *testing.T) {
	t.Run("seeds defaults and overlays the given pairs", func(t *testing.T) {
		vars, err := scaffold.ParseVarFlags(loadManifest(t), []string{"name=facturation", "entity=facture"})
		require.NoError(t, err)
		assert.Equal(t, "facturation", vars["Name"])
		assert.Equal(t, "facture", vars["Entity"])
		assert.Equal(t, true, vars["CreateTable"])
	})

	t.Run("parses a bool and accepts any key spelling", func(t *testing.T) {
		vars, err := scaffold.ParseVarFlags(loadManifest(t), []string{"create-table=false"})
		require.NoError(t, err)
		assert.Equal(t, false, vars["CreateTable"])

		vars, err = scaffold.ParseVarFlags(loadManifest(t), []string{"CreateTable=false"})
		require.NoError(t, err)
		assert.Equal(t, false, vars["CreateTable"])
	})

	t.Run("rejects an unknown key", func(t *testing.T) {
		_, err := scaffold.ParseVarFlags(loadManifest(t), []string{"colour=red"})
		assert.ErrorContains(t, err, "colour")
	})

	t.Run("rejects a bad bool", func(t *testing.T) {
		_, err := scaffold.ParseVarFlags(loadManifest(t), []string{"create-table=maybe"})
		assert.ErrorContains(t, err, "create-table")
	})

	t.Run("rejects a pair without =", func(t *testing.T) {
		_, err := scaffold.ParseVarFlags(loadManifest(t), []string{"name"})
		assert.ErrorContains(t, err, "key=value")
	})
}

func TestMissingRequired(t *testing.T) {
	t.Run("lists required inputs that are empty", func(t *testing.T) {
		vars, err := scaffold.ParseVarFlags(loadManifest(t), []string{"name=facturation"})
		require.NoError(t, err)
		assert.Equal(t, []string{"Entity"}, scaffold.MissingRequired(loadManifest(t), vars))
	})

	t.Run("is empty when every required input is given", func(t *testing.T) {
		vars, err := scaffold.ParseVarFlags(loadManifest(t), []string{"name=facturation", "entity=facture"})
		require.NoError(t, err)
		assert.Empty(t, scaffold.MissingRequired(loadManifest(t), vars))
	})
}

func TestValidate(t *testing.T) {
	ok := func() map[string]any { return map[string]any{"Name": "facturation", "Entity": "facture"} }

	t.Run("accepts lower-case words with digits", func(t *testing.T) {
		vars := ok()
		vars["Entity"] = "facture2"
		assert.NoError(t, scaffold.Validate(t.TempDir(), vars))
	})

	t.Run("rejects a name with a hyphen", func(t *testing.T) {
		vars := ok()
		vars["Name"] = "order-management"
		assert.ErrorContains(t, scaffold.Validate(t.TempDir(), vars), "name")
	})

	t.Run("rejects an entity with an upper-case letter", func(t *testing.T) {
		vars := ok()
		vars["Entity"] = "Facture"
		assert.ErrorContains(t, scaffold.Validate(t.TempDir(), vars), "entity")
	})

	t.Run("rejects an entity that collides with a generated identifier", func(t *testing.T) {
		vars := ok()
		vars["Entity"] = "label"
		assert.ErrorContains(t, scaffold.Validate(t.TempDir(), vars), "reserved")
	})

	for _, e := range []string{"msg", "srv", "body", "req", "resp"} {
		t.Run("accepts entity "+e+" (no template local shadows it)", func(t *testing.T) {
			vars := ok()
			vars["Entity"] = e
			assert.NoError(t, scaffold.Validate(t.TempDir(), vars))
		})
	}

	for _, e := range []string{"gateway", "publisher", "sent", "httptest", "cmd", "adapter", "svc"} {
		t.Run("rejects reserved entity "+e, func(t *testing.T) {
			vars := ok()
			vars["Entity"] = e
			assert.ErrorContains(t, scaffold.Validate(t.TempDir(), vars), "reserved")
		})
	}

	t.Run("rejects a Go keyword as name", func(t *testing.T) {
		vars := ok()
		vars["Name"] = "func"
		assert.ErrorContains(t, scaffold.Validate(t.TempDir(), vars), "Go keyword")
	})

	t.Run("rejects a Go keyword as entity", func(t *testing.T) {
		vars := ok()
		vars["Entity"] = "type"
		assert.ErrorContains(t, scaffold.Validate(t.TempDir(), vars), "Go keyword")
	})

	t.Run("rejects a Postgres reserved word as name", func(t *testing.T) {
		vars := ok()
		vars["Name"] = "order"
		assert.ErrorContains(t, scaffold.Validate(t.TempDir(), vars), "Postgres reserved")
	})

	t.Run("rejects an entity that collides with a template import", func(t *testing.T) {
		vars := ok()
		vars["Entity"] = "command"
		assert.ErrorContains(t, scaffold.Validate(t.TempDir(), vars), "reserved")
	})

	t.Run("rejects an existing proto directory", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts", "proto", "facturation"), 0o755))
		assert.ErrorContains(t, scaffold.Validate(root, ok()), "contracts/proto/facturation")
	})

	t.Run("rejects an existing buf.gen file", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "contracts"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "contracts", "buf.gen.facturation.yaml"), nil, 0o600))
		assert.ErrorContains(t, scaffold.Validate(root, ok()), "contracts/buf.gen.facturation.yaml")
	})

	t.Run("rejects an existing module directory", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, "modules", "facturation"), 0o755))
		assert.ErrorContains(t, scaffold.Validate(root, ok()), "modules/facturation")
	})
}

func TestRenderFunc(t *testing.T) {
	t.Run("renders into the root and makes arch-probes.sh executable", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, scaffold.Render(scaffold.EmbeddedFS(), root, fixtureVars()))
		info, err := os.Stat(filepath.Join(root, "modules", "facturation", "scripts", "arch-probes.sh"))
		require.NoError(t, err)
		assert.NotZero(t, info.Mode()&0o100, "arch-probes.sh should be executable")
	})
}
