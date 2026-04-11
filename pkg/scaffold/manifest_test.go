package scaffold

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testManifestTOML = `
[variables]
name          = ""
org-prefix    = "github.com/acme"
with-connect  = true
with-database = false
license       = ["MIT", "BSD-3", "Apache"]

[conditions]
"modules/{{.Name}}/internal/adapters/inbound/connect" = "{{if .WithConnect}}true{{end}}"
"modules/{{.Name}}/internal/infra/persistence/migrations" = "{{if .WithDatabase}}true{{end}}"
`

func TestLoadManifest_ParsesVariables(t *testing.T) {
	fsys := fstest.MapFS{
		"template.toml": &fstest.MapFile{Data: []byte(testManifestTOML)},
	}

	m, err := LoadManifest(fsys)
	require.NoError(t, err)

	byName := make(map[string]Variable, len(m.Variables))
	for _, v := range m.Variables {
		byName[v.Name] = v
	}

	// text variable (required)
	require.Contains(t, byName, "Name")
	assert.Equal(t, KindText, byName["Name"].Kind)
	assert.Equal(t, "", byName["Name"].Default)

	// text variable with default
	require.Contains(t, byName, "OrgPrefix")
	assert.Equal(t, KindText, byName["OrgPrefix"].Kind)
	assert.Equal(t, "github.com/acme", byName["OrgPrefix"].Default)

	// bool true
	require.Contains(t, byName, "WithConnect")
	assert.Equal(t, KindBool, byName["WithConnect"].Kind)
	assert.Equal(t, true, byName["WithConnect"].Default)

	// bool false
	require.Contains(t, byName, "WithDatabase")
	assert.Equal(t, KindBool, byName["WithDatabase"].Kind)
	assert.Equal(t, false, byName["WithDatabase"].Default)

	// choice
	require.Contains(t, byName, "License")
	assert.Equal(t, KindChoice, byName["License"].Kind)
	assert.Equal(t, []string{"MIT", "BSD-3", "Apache"}, byName["License"].Default)
}

func TestLoadManifest_ParsesConditions(t *testing.T) {
	fsys := fstest.MapFS{
		"template.toml": &fstest.MapFile{Data: []byte(testManifestTOML)},
	}

	m, err := LoadManifest(fsys)
	require.NoError(t, err)

	assert.Equal(t,
		"{{if .WithConnect}}true{{end}}",
		m.Conditions["modules/{{.Name}}/internal/adapters/inbound/connect"],
	)
}

func TestLoadManifest_MissingFile(t *testing.T) {
	_, err := LoadManifest(fstest.MapFS{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template.toml")
}

func TestLoadManifest_InvalidTOML(t *testing.T) {
	fsys := fstest.MapFS{
		"template.toml": &fstest.MapFile{Data: []byte(`not valid toml ===`)},
	}
	_, err := LoadManifest(fsys)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse template.toml")
}
