package custom_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piprim/mmw/pkg/archtest/custom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainPurityValidator(t *testing.T) {
	t.Run("passes for clean domain package with only external imports", func(t *testing.T) {
		dir := t.TempDir()
		domainDir := filepath.Join(dir, "modules", "mymod", "internal", "domain")
		require.NoError(t, os.MkdirAll(domainDir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(domainDir, "entity.go"), []byte(`
package domain

import "github.com/google/uuid"

type ID struct{ v uuid.UUID }
`), 0600))

		v := &custom.DomainPurityValidator{ModulesDir: filepath.Join(dir, "modules")}
		assert.NoError(t, v.Check())
	})

	t.Run("fails when domain layer imports contracts package", func(t *testing.T) {
		dir := t.TempDir()
		domainDir := filepath.Join(dir, "modules", "mymod", "internal", "domain")
		require.NoError(t, os.MkdirAll(domainDir, 0755))

		require.NoError(t, os.WriteFile(filepath.Join(domainDir, "events.go"), []byte(`
package domain

import "github.com/pivaldi/mmw-contracts/go/application/todo"

type Event struct{ topic string }
`), 0600))

		v := &custom.DomainPurityValidator{ModulesDir: filepath.Join(dir, "modules")}
		err := v.Check()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mmw-contracts")
	})
}
