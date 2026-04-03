package custom_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piprim/mmw/pkg/archtest/custom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationPurityValidator_Pass(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "modules", "mymod", "internal", "application")
	require.NoError(t, os.MkdirAll(appDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(appDir, "service.go"), []byte(`
package application

import "context"

type Service interface {
	Do(ctx context.Context) error
}
`), 0600))

	v := &custom.ApplicationPurityValidator{ModulesDir: filepath.Join(dir, "modules")}
	assert.NoError(t, v.Check())
}

func TestApplicationPurityValidator_Fail_ContractsImport(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "modules", "mymod", "internal", "application")
	require.NoError(t, os.MkdirAll(appDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(appDir, "errors.go"), []byte(`
package application

import deftodo "github.com/pivaldi/mmw-contracts/definitions/todo"

type ErrorCode = deftodo.ErrorCode
`), 0600))

	v := &custom.ApplicationPurityValidator{ModulesDir: filepath.Join(dir, "modules")}
	err := v.Check()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mmw-contracts")
}
