package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"name", "Name"},
		{"with-connect", "WithConnect"},
		{"with_connect", "WithConnect"},
		{"withConnect", "WithConnect"},
		{"WithConnect", "WithConnect"},
		{"org-prefix", "OrgPrefix"},
		{"with-database", "WithDatabase"},
		{"a", "A"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, NormalizeKey(tt.in), "input: %q", tt.in)
	}
}

func TestEnrichVars(t *testing.T) {
	vars := map[string]any{
		"Name":      "payment",
		"OrgPrefix": "github.com/acme",
	}
	require.NoError(t, EnrichVars(vars))

	assert.Equal(t, "Payment", vars["NameTitle"])
	assert.Equal(t, "github.com/acme/mmw-payment", vars["ModulePath"])
	assert.Equal(t, "github.com/acme/mmw-contracts", vars["ContractsPath"])
	assert.Equal(t, "github.com/piprim/mmw", vars["PlatformPath"])
	assert.Equal(t, "defpayment", vars["PkgDef"])
}

func TestEnrichVars_MissingName(t *testing.T) {
	err := EnrichVars(map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

func TestEnrichVars_PreservesPlatformPath(t *testing.T) {
	vars := map[string]any{
		"Name":         "payment",
		"OrgPrefix":    "github.com/acme",
		"PlatformPath": "github.com/myorg/mmw",
	}
	require.NoError(t, EnrichVars(vars))
	assert.Equal(t, "github.com/myorg/mmw", vars["PlatformPath"])
}
