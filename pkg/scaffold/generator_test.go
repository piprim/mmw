package scaffold_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateModule_Minimal(t *testing.T) {
	dir := t.TempDir()
	vars := map[string]any{
		"Name":         "payment",
		"OrgPrefix":    "github.com/acme",
		"WithConnect":  false,
		"WithContract": false,
		"WithDatabase": false,
	}
	require.NoError(t, scaffold.EnrichVars(vars))

	require.NoError(t, scaffold.GenerateModule(scaffold.EmbeddedFS(), dir, vars))

	// Base files always present
	assertFileExists(t, dir, "modules/payment/go.mod")
	assertFileExists(t, dir, "modules/payment/paymentmod.go")
	assertFileExists(t, dir, "modules/payment/mise.toml")
	assertFileContains(t, dir, "modules/payment/go.mod", "module github.com/acme/mmw-payment")
	assertFileContains(t, dir, "modules/payment/paymentmod.go", "package payment")
	assertFileContains(t, dir, "modules/payment/paymentmod.go", "type Module struct")
	assertFileContains(t, dir, "modules/payment/paymentmod.go", "type Infrastructure struct")

	// Domain layer
	assertFileExists(t, dir, "modules/payment/internal/domain/payment.go")
	assertFileContains(t, dir, "modules/payment/internal/domain/payment.go", "type Payment struct")
	assertFileExists(t, dir, "modules/payment/internal/domain/events.go")
	assertFileContains(t, dir, "modules/payment/internal/domain/events.go", `EventTypeCreated = "payment.created"`)
	assertFileExists(t, dir, "modules/payment/internal/domain/errors.go")
	assertFileExists(t, dir, "modules/payment/internal/domain/value_objects.go")

	// Application layer
	assertFileExists(t, dir, "modules/payment/internal/application/service.go")
	assertFileContains(t, dir, "modules/payment/internal/application/service.go", "type PaymentService interface")
	assertFileExists(t, dir, "modules/payment/internal/application/errors.go")
	assertFileContains(t, dir, "modules/payment/internal/application/errors.go", "type ErrorCode int")
	assertFileExists(t, dir, "modules/payment/internal/application/ports/ports.go")
	assertFileContains(t, dir, "modules/payment/internal/application/ports/ports.go", "type PaymentRepository interface")

	// Adapters
	assertFileExists(t, dir, "modules/payment/internal/adapters/outbound/persistence/postgres/repository.go")
	assertFileExists(t, dir, "modules/payment/internal/adapters/outbound/events/topics.go")
	assertFileExists(t, dir, "modules/payment/internal/adapters/outbound/events/outbox_dispatcher.go")
	assertFileExists(t, dir, "modules/payment/internal/infra/config/config.go")

	// Connect adapter NOT present (WithConnect: false)
	assertFileNotExists(t, dir, "modules/payment/internal/adapters/inbound/connect/handler.go")

	// Migration NOT present (WithDatabase: false)
	assertFileNotExists(t, dir, "modules/payment/internal/infra/persistence/migrations/migrations.go")

	// Contract NOT present (WithContract: false)
	assertFileNotExists(t, dir, "contracts/go/application/payment/api.go")
}

func TestGenerateModule_WithAllOptions(t *testing.T) {
	dir := t.TempDir()
	vars := map[string]any{
		"Name":         "billing",
		"OrgPrefix":    "github.com/acme",
		"WithConnect":  true,
		"WithContract": true,
		"WithDatabase": true,
	}
	require.NoError(t, scaffold.EnrichVars(vars))

	require.NoError(t, scaffold.GenerateModule(scaffold.EmbeddedFS(), dir, vars))

	// Connect adapter present
	assertFileExists(t, dir, "modules/billing/internal/adapters/inbound/connect/handler.go")
	assertFileExists(t, dir, "modules/billing/internal/adapters/inbound/connect/errors.go")
	assertFileContains(t, dir, "modules/billing/internal/adapters/inbound/connect/handler.go", "type BillingHandler struct")

	// inproc adapter present (always when WithContract)
	assertFileExists(t, dir, "modules/billing/internal/adapters/inbound/inproc/adapter.go")
	assertFileContains(t, dir, "modules/billing/internal/adapters/inbound/inproc/adapter.go", "var _ defbilling.BillingService")

	// Migration present
	assertFileExists(t, dir, "modules/billing/internal/infra/persistence/migrations/migrations.go")
	assertFileExists(t, dir, "modules/billing/cmd/migration/main.go")

	// Contract present
	assertFileExists(t, dir, "contracts/go/application/billing/api.go")
	assertFileExists(t, dir, "contracts/go/application/billing/dto.go")
	assertFileExists(t, dir, "contracts/go/application/billing/errors.go")
	assertFileExists(t, dir, "contracts/go/application/billing/inproc_client.go")
	assertFileContains(t, dir, "contracts/go/application/billing/api.go", "type BillingService interface")

	// Proto present (WithConnect + WithContract)
	assertFileExists(t, dir, "contracts/proto/billing/v1/billing.proto")
	assertFileContains(t, dir, "contracts/proto/billing/v1/billing.proto", "service BillingService")
}

func TestGenerateModule_WithContractNoConnect(t *testing.T) {
	dir := t.TempDir()
	vars := map[string]any{
		"Name":         "inventory",
		"OrgPrefix":    "github.com/acme",
		"WithConnect":  false,
		"WithContract": true,
		"WithDatabase": false,
	}
	require.NoError(t, scaffold.EnrichVars(vars))

	require.NoError(t, scaffold.GenerateModule(scaffold.EmbeddedFS(), dir, vars))

	// Contract present
	assertFileExists(t, dir, "contracts/go/application/inventory/api.go")

	// Proto NOT present (no Connect)
	assertFileNotExists(t, dir, "contracts/proto/inventory/v1/inventory.proto")

	// Connect handler NOT present
	assertFileNotExists(t, dir, "modules/inventory/internal/adapters/inbound/connect/handler.go")

	// inproc adapter present
	assertFileExists(t, dir, "modules/inventory/internal/adapters/inbound/inproc/adapter.go")
}

func TestGenerateContract(t *testing.T) {
	dir := t.TempDir()
	vars := map[string]any{
		"Name":         "shipping",
		"OrgPrefix":    "github.com/acme",
		"WithConnect":  true,
		"WithContract": true,
		"WithDatabase": false,
	}
	require.NoError(t, scaffold.EnrichVars(vars))

	require.NoError(t, scaffold.GenerateContract(scaffold.EmbeddedFS(), dir, vars))

	assertFileExists(t, dir, "contracts/go/application/shipping/api.go")
	assertFileExists(t, dir, "contracts/go/application/shipping/dto.go")
	assertFileExists(t, dir, "contracts/go/application/shipping/errors.go")
	assertFileExists(t, dir, "contracts/go/application/shipping/inproc_client.go")
	assertFileExists(t, dir, "contracts/proto/shipping/v1/shipping.proto")
}

func TestGenerateModule_EmptyName(t *testing.T) {
	err := scaffold.GenerateModule(scaffold.EmbeddedFS(), t.TempDir(), map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

// helpers

func assertFileExists(t *testing.T, base, rel string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(base, rel))
	assert.NoError(t, err, "expected file to exist: %s", rel)
}

func assertFileNotExists(t *testing.T, base, rel string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(base, rel))
	assert.True(t, os.IsNotExist(err), "expected file NOT to exist: %s", rel)
}

func assertFileContains(t *testing.T, base, rel, substr string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(base, rel))
	require.NoError(t, err, "reading %s", rel)
	assert.Contains(t, string(content), substr, "file %s should contain %q", rel, substr)
}
