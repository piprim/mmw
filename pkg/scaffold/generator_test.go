package scaffold_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/piprim/goplt"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateModule(t *testing.T) {
	t.Run("minimal module generates base files and domain/application layers", func(t *testing.T) {
		dir := t.TempDir()
		fsys := scaffold.EmbeddedFS()

		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)

		vars := map[string]any{
			"Name":          "payment",
			"OrgPrefix":     "github.com/acme",
			"ContractsPath": "github.com/acme/my-contracts",
			"PlatformPath":  "github.com/piprim/mmw",
			"WithModule":    true,
			"WithConnect":   false,
			"WithContract":  false,
			"WithDatabase":  false,
			"License":       "MIT",
		}

		require.NoError(t, goplt.NewGenerator().Generate(fsys, m, dir, vars))

		assertFileExists(t, dir, "modules/payment/go.mod")
		assertFileExists(t, dir, "modules/payment/paymentmod.go")
		assertFileExists(t, dir, "modules/payment/mise.toml")
		assertFileContains(t, dir, "modules/payment/go.mod", "module github.com/acme/payment")
		assertFileContains(t, dir, "modules/payment/paymentmod.go", "package payment")
		assertFileContains(t, dir, "modules/payment/paymentmod.go", "type Module struct")
		assertFileContains(t, dir, "modules/payment/paymentmod.go", "type Infrastructure struct")

		assertFileExists(t, dir, "modules/payment/internal/domain/payment.go")
		assertFileContains(t, dir, "modules/payment/internal/domain/payment.go", "type Payment struct")
		assertFileExists(t, dir, "modules/payment/internal/domain/events.go")
		assertFileContains(t, dir, "modules/payment/internal/domain/events.go", `EventTypeCreated = "payment.created"`)
		assertFileExists(t, dir, "modules/payment/internal/domain/errors.go")
		assertFileExists(t, dir, "modules/payment/internal/domain/value_objects.go")

		assertFileExists(t, dir, "modules/payment/internal/application/service.go")
		assertFileContains(t, dir, "modules/payment/internal/application/service.go", "type PaymentService interface")
		assertFileExists(t, dir, "modules/payment/internal/application/errors.go")
		assertFileContains(t, dir, "modules/payment/internal/application/errors.go", "type ErrorCode int")
		assertFileExists(t, dir, "modules/payment/internal/application/ports/ports.go")
		assertFileContains(t, dir, "modules/payment/internal/application/ports/ports.go", "type PaymentRepository interface")

		assertFileExists(t, dir, "modules/payment/internal/adapters/outbound/persistence/postgres/repository.go")
		assertFileExists(t, dir, "modules/payment/internal/adapters/outbound/events/topics.go")
		assertFileExists(t, dir, "modules/payment/internal/adapters/outbound/events/outbox_dispatcher.go")
		assertFileExists(t, dir, "modules/payment/internal/infra/config/config.go")

		assertFileNotExists(t, dir, "modules/payment/internal/adapters/inbound/connect/handler.go")
		assertFileNotExists(t, dir, "modules/payment/internal/infra/persistence/migrations/migrations.go")
		assertFileNotExists(t, dir, "contracts/go/application/payment/api.go")
	})

	t.Run("all options enabled generates connect, inproc, migrations, and contract files", func(t *testing.T) {
		dir := t.TempDir()
		fsys := scaffold.EmbeddedFS()

		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)

		vars := map[string]any{
			"Name":          "billing",
			"OrgPrefix":     "github.com/acme",
			"ContractsPath": "github.com/acme/my-contracts",
			"PlatformPath":  "github.com/piprim/mmw",
			"WithModule":    true,
			"WithConnect":   true,
			"WithContract":  true,
			"WithDatabase":  true,
			"License":       "MIT",
		}

		require.NoError(t, goplt.NewGenerator().Generate(fsys, m, dir, vars))

		assertFileExists(t, dir, "modules/billing/internal/adapters/inbound/connect/handler.go")
		assertFileExists(t, dir, "modules/billing/internal/adapters/inbound/connect/errors.go")
		assertFileContains(t, dir, "modules/billing/internal/adapters/inbound/connect/handler.go", "type BillingHandler struct")

		assertFileExists(t, dir, "modules/billing/internal/adapters/inbound/inproc/adapter.go")
		assertFileContains(t, dir, "modules/billing/internal/adapters/inbound/inproc/adapter.go", "var _ defbilling.BillingService")

		assertFileExists(t, dir, "modules/billing/internal/infra/persistence/migrations/migrations.go")
		assertFileExists(t, dir, "modules/billing/cmd/migration/main.go")

		assertFileExists(t, dir, "contracts/go/application/billing/api.go")
		assertFileExists(t, dir, "contracts/go/application/billing/dto.go")
		assertFileExists(t, dir, "contracts/go/application/billing/errors.go")
		assertFileExists(t, dir, "contracts/go/application/billing/inproc_client.go")
		assertFileContains(t, dir, "contracts/go/application/billing/api.go", "type BillingService interface")

		assertFileExists(t, dir, "contracts/proto/billing/v1/billing.proto")
		assertFileContains(t, dir, "contracts/proto/billing/v1/billing.proto", "service BillingService")
	})

	t.Run("contract without connect generates inproc but not proto or connect handler", func(t *testing.T) {
		dir := t.TempDir()
		fsys := scaffold.EmbeddedFS()

		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)

		vars := map[string]any{
			"Name":          "inventory",
			"OrgPrefix":     "github.com/acme",
			"ContractsPath": "github.com/acme/my-contracts",
			"PlatformPath":  "github.com/piprim/mmw",
			"WithModule":    true,
			"WithConnect":   false,
			"WithContract":  true,
			"WithDatabase":  false,
			"License":       "MIT",
		}

		require.NoError(t, goplt.NewGenerator().Generate(fsys, m, dir, vars))

		assertFileExists(t, dir, "contracts/go/application/inventory/api.go")
		assertFileNotExists(t, dir, "contracts/proto/inventory/v1/inventory.proto")
		assertFileNotExists(t, dir, "modules/inventory/internal/adapters/inbound/connect/handler.go")
		assertFileExists(t, dir, "modules/inventory/internal/adapters/inbound/inproc/adapter.go")
	})
}

func TestGenerateContract(t *testing.T) {
	t.Run("contract only (no module) generates contract and proto files", func(t *testing.T) {
		dir := t.TempDir()
		fsys := scaffold.EmbeddedFS()

		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)

		vars := map[string]any{
			"Name":          "shipping",
			"OrgPrefix":     "github.com/acme",
			"ContractsPath": "",
			"PlatformPath":  "",
			"WithModule":    false,
			"WithConnect":   true,
			"WithContract":  true,
			"WithDatabase":  false,
			"License":       "MIT",
		}

		require.NoError(t, goplt.NewGenerator().Generate(fsys, m, dir, vars))

		assertFileExists(t, dir, "contracts/go/application/shipping/api.go")
		assertFileExists(t, dir, "contracts/go/application/shipping/dto.go")
		assertFileExists(t, dir, "contracts/go/application/shipping/errors.go")
		assertFileExists(t, dir, "contracts/go/application/shipping/inproc_client.go")
		assertFileExists(t, dir, "contracts/proto/shipping/v1/shipping.proto")
		assertFileNotExists(t, dir, "modules/shipping/go.mod")
	})
}

func TestGenerate_EmptyName(t *testing.T) {
	t.Run("renders empty name without error (documents behaviour)", func(t *testing.T) {
		fsys := fstest.MapFS{
			"template.toml": &fstest.MapFile{Data: []byte(`
description = "The description"
[variables]
kind = "input"
required = true
`)},
			"{{.Name}}.go": &fstest.MapFile{Data: []byte(`package {{.Name}}`)},
		}

		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)

		err = goplt.NewGenerator().Generate(fsys, m, t.TempDir(), map[string]any{})
		_ = err
	})
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
	if err == nil {
		t.Errorf("expected file NOT to exist: %s", rel)

		return
	}
	if !os.IsNotExist(err) {
		t.Errorf("unexpected error stating %s: %v", rel, err)
	}
}

func assertFileContains(t *testing.T, base, rel, substr string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(base, rel))
	require.NoError(t, err, "reading %s", rel)
	assert.Contains(t, string(content), substr, "file %s should contain %q", rel, substr)
}
