package scaffold_test

import (
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/piprim/goplt"
	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureVars are the variables every scaffold test renders with.
func fixtureVars() map[string]any {
	return map[string]any{
		"Name":            "facturation",
		"Entity":          "facture",
		"CreateTable":     true,
		"Database":        true,
		"Shape":           "aggregate",
		"Grpc":            true,
		"ModulePath":      "ovya.fr/rcv/facturation",
		"OrgPrefix":       "ovya.fr/rcv",
		"ContractsPath":   "ovya.fr/rcv/contracts",
		"PlatformPath":    "github.com/piprim/mmw",
		"TimestampCreate": "20261013120100",
		"TimestampSeed":   "20261013120200",
	}
}

// renderBoth renders the module tree then the slice tree into dir.
func renderBoth(dir string, vars map[string]any) error {
	for _, fsys := range []fs.FS{scaffold.ModuleFS(), scaffold.SliceFS()} {
		m, err := goplt.LoadManifest(fsys)
		if err != nil {
			return err
		}
		if err := goplt.NewGenerator().Generate(fsys, m, dir, vars); err != nil {
			return err
		}
	}

	return nil
}

// renderWith renders the embedded templates into a temp dir and returns it.
func renderWith(t *testing.T, vars map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, renderBoth(dir, vars))

	return dir
}

var fixtureDir string

// renderFixture renders the fixture once per test binary.
func renderFixture(t *testing.T) string {
	t.Helper()
	if fixtureDir != "" {
		return fixtureDir
	}
	dir, err := os.MkdirTemp("", "scaffold-fixture-")
	require.NoError(t, err)
	require.NoError(t, renderBoth(dir, fixtureVars()))
	fixtureDir = dir

	return dir
}

func TestMain(m *testing.M) {
	code := m.Run()
	if fixtureDir != "" {
		_ = os.RemoveAll(fixtureDir)
	}
	os.Exit(code)
}

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

func assertFileNotContains(t *testing.T, base, rel, substr string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(base, rel))
	require.NoError(t, err, "reading %s", rel)
	assert.NotContains(t, string(content), substr, "file %s should not contain %q", rel, substr)
}

func TestRender(t *testing.T) {
	dir := renderFixture(t)
	mod := "modules/facturation"

	t.Run("no rendered file keeps a template action", func(t *testing.T) {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			content, err := os.ReadFile(path)
			require.NoError(t, err)
			rel, _ := filepath.Rel(dir, path)
			assert.NotContains(t, string(content), "[[.", "unrendered action in %s", rel)
			assert.NotContains(t, rel, "[[", "unrendered action in path %s", rel)

			return nil
		})
		require.NoError(t, err)
	})

	t.Run("mise.toml keeps its own template actions", func(t *testing.T) {
		assertFileContains(t, dir, mod+"/mise.toml", "{{config_root}}")
	})

	t.Run("domain layer", func(t *testing.T) {
		for _, f := range []string{
			"internal/domain/event.go",
			"internal/domain/facture/facture.go",
			"internal/domain/facture/snapshot.go",
			"internal/domain/facture/value_objects.go",
			"internal/domain/facture/events.go",
			"internal/domain/facture/errors.go",
			"internal/domain/facture/facture_test.go",
			"internal/domain/facture/value_objects_test.go",
		} {
			assertFileExists(t, dir, mod+"/"+f)
		}
		assertFileContains(t, dir, mod+"/internal/domain/facture/facture.go", "package facture")
		assertFileContains(t, dir, mod+"/internal/domain/facture/facture.go", "type Facture struct")
		assertFileContains(t, dir, mod+"/internal/domain/facture/events.go", `EventTypeDeleted = "facture.deleted"`)
		assertFileContains(t, dir, mod+"/internal/domain/facture/facture.go", `"ovya.fr/rcv/facturation/internal/domain"`)
	})

	t.Run("application layer", func(t *testing.T) {
		for _, f := range []string{
			"internal/application/ports/facture_repository.go",
			"internal/application/ports/uow.go",
			"internal/application/ports/events.go",
			"internal/application/query/facture.go",
			"internal/application/query/facture_test.go",
			"internal/application/command/facture_delete.go",
			"internal/application/command/facture_delete_test.go",
			"internal/application/facture.go",
			"internal/testhelpers/fakes.go",
			"internal/testhelpers/facture_fake.go",
		} {
			assertFileExists(t, dir, mod+"/"+f)
		}
		assertFileContains(t, dir, mod+"/internal/application/facture.go", "type FactureService interface")
		assertFileContains(t, dir, mod+"/internal/application/ports/facture_repository.go", "FactureFindByID(ctx context.Context, id facture.ID)")
		assertFileContains(t, dir, mod+"/internal/application/command/facture_delete.go", "WithTransaction(ctx")
		assertFileContains(t, dir, mod+"/internal/testhelpers/facture_fake.go", "map[facture.ID]")
	})

	t.Run("inbound adapters", func(t *testing.T) {
		for _, f := range []string{
			"internal/adapters/inbound/connect/facture_handler.go",
			"internal/adapters/inbound/connect/facture_handler_test.go",
			"internal/adapters/inbound/connect/errors.go",
			"internal/adapters/inbound/connect/errors_test.go",
			"internal/adapters/inbound/connect/auth_middleware.go",
			"internal/adapters/inbound/connect/auth_middleware_test.go",
			"internal/adapters/inbound/mapper/facture.go",
			"internal/adapters/inbound/mapper/facture_test.go",
			"internal/adapters/inbound/mapper/errors.go",
			"internal/adapters/inbound/mapper/errors_test.go",
			"internal/adapters/inbound/mapper/facture_errors.go",
			"internal/adapters/inbound/mapper/facture_errors_test.go",
			"internal/adapters/inbound/inproc/facture_contract_adapter.go",
		} {
			assertFileExists(t, dir, mod+"/"+f)
		}
		assertFileContains(t, dir, mod+"/internal/adapters/inbound/inproc/facture_contract_adapter.go", "var _ facturationdef.FactureService = (*FactureContractAdapter)(nil)")
		assertFileContains(t, dir, mod+"/internal/adapters/inbound/connect/facture_handler.go", `facturationv1 "ovya.fr/rcv/contracts/go/network/facturation/v1"`)
	})

	t.Run("outbound adapters", func(t *testing.T) {
		assertFileContains(t, dir, mod+"/internal/adapters/outbound/events/facture_events_test.go", "domainTopics[factureEventType]")
		for _, f := range []string{
			"internal/adapters/outbound/persistence/postgres/facture_repository.go",
			"internal/adapters/outbound/persistence/postgres/facture_repository_test.go",
			"internal/adapters/outbound/persistence/postgres/facture_row.go",
			"internal/adapters/outbound/persistence/postgres/facture_row_test.go",
			"internal/adapters/outbound/persistence/postgres/README.md",
			"internal/adapters/outbound/persistence/postgres/dao/boamodel/models.go",
			"internal/adapters/outbound/persistence/postgres/dao/boamodel/models.go.tmpl",
			"internal/adapters/outbound/persistence/postgres/dao/boamodel/columns.go",
			"internal/adapters/outbound/persistence/postgres/dao/boamodel/columns.go.tmpl",
			"internal/adapters/outbound/persistence/postgres/dao/boamodel/q.go",
			"internal/adapters/outbound/persistence/postgres/dao/boamodel/q.go.tmpl",
			"internal/adapters/outbound/persistence/postgres/dao/sqlc/query.sql",
			"internal/adapters/outbound/persistence/postgres/dao/sqlc/presence_args.go",
			"internal/adapters/outbound/events/outbox_dispatcher.go",
			"internal/adapters/outbound/events/topics.go",
			"internal/adapters/outbound/events/topics_test.go",
			"internal/adapters/outbound/events/facture_events.go",
			"internal/adapters/outbound/events/facture_events_test.go",
			"internal/adapters/outbound/events/bus_publisher.go",
		} {
			assertFileExists(t, dir, mod+"/"+f)
		}
		boamodel := mod + "/internal/adapters/outbound/persistence/postgres/dao/boamodel/"
		assertFileContains(t, dir, boamodel+"models.go", "type FacturationFacture struct")
		assertFileContains(t, dir, boamodel+"models.go", `boa:"table:facturation.event,alias:t"`)
		assertFileContains(t, dir, boamodel+"models.go.tmpl", "{{range $model := .Entities}}")
		assertFileContains(t, dir, mod+"/internal/adapters/outbound/events/facture_events.go", "factureTopic     = facturationdef.TopicFactureDeleted")
	})

	t.Run("entity equal to module name renders", func(t *testing.T) {
		vars := fixtureVars()
		vars["Entity"] = "facturation"
		same := renderWith(t, vars)
		assertFileExists(t, same, "modules/facturation/internal/domain/facturation/facturation.go")
		assertFileContains(t, same, "modules/facturation/internal/domain/facturation/facturation.go", "package facturation")
	})
	t.Run("boa:generate runs the repository's bungen fork", func(t *testing.T) {
		// Upstream bungen on PATH ignores --template and --presence.
		assertFileContains(t, dir, mod+"/mise.toml", "go run {{config_root}}/../../tools/bungen/main.go model")
		assertFileNotContains(t, dir, mod+"/mise.toml", "\n  bungen model")
	})

	t.Run("standalone commands check the database password with RequireSecret", func(t *testing.T) {
		for _, f := range []string{"/cmd/facturation/main.go", "/cmd/migration/migrate.go"} {
			assertFileContains(t, dir, mod+f, "RequireSecret(")
			assertFileNotContains(t, dir, mod+f, `Password == ""`)
		}
	})

	t.Run("cmd/migration does not import the module root", func(t *testing.T) {
		// The root imports the postgres adapter, which does not build until
		// boa:generate has run on the migrated table.
		assertFileNotContains(t, dir, mod+"/cmd/migration/migrate.go", `"ovya.fr/rcv/facturation"`)
		assertFileContains(t, dir, mod+"/cmd/migration/migrate.go", "migrations.Schema")
		assertFileContains(t, dir, mod+"/internal/infra/persistence/migrations/migrations.go", `const Schema = "facturation"`)
		assertFileContains(t, dir, mod+"/facturation.go", "PGSchema       = migrations.Schema")
	})

	t.Run("infra, root and tooling", func(t *testing.T) {
		for _, f := range []string{
			"facturation.go",
			"go.mod",
			"arch-go.yml", "sqlc.yaml", "buf.yaml", ".mockery.yaml", ".air.toml", ".envrc.example", ".gitignore", "README.md",
			"scripts/arch-probes.sh",
			"cmd/facturation/main.go", "cmd/migration/migrate.go", "cmd/migration/config.go",
			"internal/infra/config/config.go", "internal/infra/config/config_test.go", "internal/infra/config/configs/default.toml",
			"internal/infra/persistence/migrations/migrations.go",
			"internal/infra/persistence/migrations/scripts/20261013120000_create-outbox.sql",
			"internal/infra/persistence/migrations/scripts/20261013120100_create-facture.sql",
			"internal/infra/persistence/migrations/scripts/20261013120200_seed-facture.sql",
		} {
			assertFileExists(t, dir, mod+"/"+f)
		}
		assertFileContains(t, dir, mod+"/go.mod", "module ovya.fr/rcv/facturation")
		assertFileContains(t, dir, mod+"/go.mod", "replace ovya.fr/rcv/contracts => ../../contracts")
		assertFileContains(t, dir, mod+"/facturation.go", "package facturation")
		assertFileContains(t, dir, mod+"/arch-go.yml", "package: 'ovya.fr/rcv/facturation'")
		assertFileContains(t, dir, mod+"/scripts/arch-probes.sh", "mod=ovya.fr/rcv/facturation")
		assertFileNotExists(t, dir, mod+"/go.mod.tmpl")
	})

	t.Run("create-table=false drops the table and seed migrations", func(t *testing.T) {
		vars := fixtureVars()
		vars["CreateTable"] = false
		noTable := renderWith(t, vars)
		scripts := "modules/facturation/internal/infra/persistence/migrations/scripts/"
		assertFileExists(t, noTable, scripts+"20261013120000_create-outbox.sql")
		assertFileNotExists(t, noTable, scripts+"20261013120100_create-facture.sql")
		assertFileNotExists(t, noTable, scripts+"20261013120200_seed-facture.sql")
	})

	t.Run("module files carry the markers", func(t *testing.T) {
		for rel, markers := range map[string][]string{
			"facturation.go": {scaffold.MarkerField, scaffold.MarkerWire, scaffold.MarkerHandler},
			"internal/adapters/outbound/events/topics.go": {scaffold.MarkerTopic},
			"internal/adapters/outbound/events/proto.go":  {scaffold.MarkerEvent},
			"internal/adapters/inbound/mapper/errors.go":  {scaffold.MarkerError},
		} {
			for _, marker := range markers {
				t.Run(rel+" "+marker, func(t *testing.T) { assertFileContains(t, dir, mod+"/"+rel, marker) })
			}
		}
	})

	t.Run("module files name no entity", func(t *testing.T) {
		for _, rel := range []string{
			mod + "/facturation.go",
			mod + "/internal/adapters/outbound/events/topics.go",
			mod + "/internal/adapters/outbound/events/topics_test.go",
			mod + "/internal/adapters/outbound/events/proto.go",
			mod + "/internal/adapters/outbound/events/bus_publisher_test.go",
			mod + "/internal/adapters/inbound/mapper/errors.go",
			mod + "/internal/adapters/inbound/mapper/errors_test.go",
			mod + "/internal/testhelpers/fakes.go",
			mod + "/test/contract/helpers_test.go",
			"contracts/proto/facturation/v1/errors.proto",
			mod + "/internal/adapters/outbound/persistence/postgres/dao/sqlc/query.sql",
			mod + "/mise.toml",
			mod + "/internal/infra/config/config.go",
			mod + "/internal/infra/config/config_test.go",
			mod + "/internal/infra/config/configs/default.toml",
			mod + "/scripts/arch-probes.sh",
			mod + "/README.md",
			mod + "/internal/adapters/outbound/persistence/postgres/README.md",
		} {
			t.Run(rel+" lower", func(t *testing.T) { assertFileNotContains(t, dir, rel, "facture") })
			t.Run(rel+" pascal", func(t *testing.T) { assertFileNotContains(t, dir, rel, "Facture") })
		}
	})

	t.Run("slice files exist", func(t *testing.T) {
		for _, rel := range []string{
			mod + "/facture_wiring.go",
			mod + "/internal/adapters/outbound/events/facture_events.go",
			mod + "/internal/adapters/inbound/mapper/facture_errors.go",
			mod + "/internal/adapters/inbound/inproc/facture_contract_adapter.go",
			mod + "/internal/testhelpers/facture_fake.go",
			mod + "/test/contract/facture_test.go",
			"contracts/proto/facturation/v1/errors.proto",
		} {
			t.Run(rel, func(t *testing.T) { assertFileExists(t, dir, rel) })
		}
		t.Run("no generic contract_adapter.go", func(t *testing.T) {
			assertFileNotExists(t, dir, mod+"/internal/adapters/inbound/inproc/contract_adapter.go")
		})
	})

	t.Run("contracts", func(t *testing.T) {
		proto := "contracts/proto/facturation/v1/facture.proto"
		assertFileExists(t, dir, proto)
		assertFileContains(t, dir, proto, "package facturation.v1;")
		assertFileContains(t, dir, proto, "service FactureService {")
		assertFileContains(t, dir, proto, `option (options.v1.topic) = "facturation.facture.deleted.v1";`)
		assertFileNotContains(t, dir, proto, "enum ")
		errs := "contracts/proto/facturation/v1/errors.proto"
		assertFileContains(t, dir, errs, "package facturation.v1;")
		assertFileContains(t, dir, errs, "enum FacturationErrorCode {")
		assertFileContains(t, dir, errs, "FACTURATION_ERROR_CODE_INVALID_LABEL = 3;")
		assertFileContains(t, dir, errs, "FACTURATION_ERROR_CODE_GATEWAY_UNAVAILABLE = 4;")
		assertFileExists(t, dir, "contracts/buf.gen.facturation.yaml")
		assertFileContains(t, dir, "contracts/buf.gen.facturation.yaml", "- proto/facturation")
		assertFileExists(t, dir, mod+"/test/contract/helpers_test.go")
		assertFileExists(t, dir, mod+"/test/contract/facture_test.go")
		assertFileNotExists(t, dir, mod+"/test/contract/contract_test.go")
		assertFileNotExists(t, dir, "contracts/.keep")
	})
}

// shape is one row of the render matrix.
type shape struct {
	shape          string
	database, grpc bool
	name, entity   string
}

func (s shape) String() string {
	return fmt.Sprintf("%s/%s shape=%s database=%t grpc=%t", s.name, s.entity, s.shape, s.database, s.grpc)
}

func (s shape) vars() map[string]any {
	return map[string]any{
		"Name": s.name, "Entity": s.entity, "CreateTable": true,
		"Database": s.database, "Grpc": s.grpc, "Shape": s.shape,
		"ModulePath": "ovya.fr/rcv/" + s.name,
		"OrgPrefix":  "ovya.fr/rcv", "ContractsPath": "ovya.fr/rcv/contracts", "PlatformPath": "github.com/piprim/mmw",
		"TimestampCreate": "20261013120100", "TimestampSeed": "20261013120200",
	}
}

var shapes = []shape{
	{"aggregate", true, true, "facturation", "facture"},
	{"aggregate", true, false, "facturation", "facture"},
	{"read-model", true, true, "facturation", "avoir"},
	{"read-model", true, false, "facturation", "avoir"},
	{"gateway", true, true, "facturation", "facture"},
	{"gateway", true, false, "facturation", "facture"},
	{"gateway", false, true, "messagerie", "sms"},
	{"gateway", false, false, "messagerie", "sms"},
	{"gateway", false, false, "sms", "sms"},
}

// assertGofmt fails on any rendered .go file gofmt would change.
func assertGofmt(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		src, err := os.ReadFile(path)
		require.NoError(t, err)
		formatted, err := format.Source(src)
		rel, _ := filepath.Rel(dir, path)
		require.NoError(t, err, "gofmt %s", rel)
		assert.Equal(t, string(formatted), string(src), "not gofmt-clean: %s", rel)

		return nil
	})
	require.NoError(t, err)
}

// assertNoLeftoverActions fails on any rendered file or path still holding "[[".
func assertNoLeftoverActions(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		rel, _ := filepath.Rel(dir, path)
		assert.NotContains(t, string(content), "[[", "unrendered action in %s", rel)
		assert.NotContains(t, rel, "[[", "unrendered action in path %s", rel)

		return nil
	})
	require.NoError(t, err)
}

func TestRenderMatrix(t *testing.T) {
	for _, s := range shapes {
		t.Run(s.String(), func(t *testing.T) {
			dir := renderWith(t, s.vars())
			assertNoLeftoverActions(t, dir)
			assertGofmt(t, dir)
			mod := "modules/" + s.name
			pascal := strings.ToUpper(s.entity[:1]) + s.entity[1:]
			if !s.database {
				assertFileNotExists(t, dir, mod+"/internal/infra/persistence/migrations/scripts/20261013120000_create-outbox.sql")
				assertFileNotExists(t, dir, mod+"/internal/infra/persistence/migrations/scripts/20261013120100_create-"+s.entity+".sql")
			}
			if s.shape == "read-model" {
				dom := mod + "/internal/domain/avoir/"
				rm := func(name string, check func(t *testing.T)) { t.Run(name, check) }
				rm("read-model has no snapshot.go", func(t *testing.T) { assertFileNotExists(t, dir, dom+"snapshot.go") })
				rm("read-model has no events.go", func(t *testing.T) { assertFileNotExists(t, dir, dom+"events.go") })
				rm("read-model has no command package", func(t *testing.T) {
					assertFileNotExists(t, dir, mod+"/internal/application/command")
				})
				rm("read-model has no avoir_events.go", func(t *testing.T) {
					assertFileNotExists(t, dir, mod+"/internal/adapters/outbound/events/avoir_events.go")
				})
				rm("read-model has no avoir_events_test.go", func(t *testing.T) {
					assertFileNotExists(t, dir, mod+"/internal/adapters/outbound/events/avoir_events_test.go")
				})
				rm("read-model entity has no Delete", func(t *testing.T) { assertFileNotContains(t, dir, dom+"avoir.go", "Delete(") })
				rm("read-model keeps exported fields", func(t *testing.T) {
					assertFileContains(t, dir, dom+"avoir.go", "Label     Label")
				})
				rm("read-model keeps a row mapper", func(t *testing.T) {
					assertFileContains(t, dir, mod+"/internal/adapters/outbound/persistence/postgres/avoir_row.go", "ReconstituteLabel(")
				})
				rm("read-model wiring builds the app service from the repo", func(t *testing.T) {
					assertFileContains(t, dir, mod+"/avoir_wiring.go", "application.NewAvoirAppService(repo)")
				})
				avoirProto := "contracts/proto/facturation/v1/avoir.proto"
				rm("read-model proto has GetAvoir", func(t *testing.T) { assertFileContains(t, dir, avoirProto, "rpc GetAvoir") })
				rm("read-model proto has no DeletedEvent", func(t *testing.T) {
					assertFileNotContains(t, dir, avoirProto, "DeletedEvent")
				})
				rm("read-model proto has no DeleteAvoir", func(t *testing.T) {
					assertFileNotContains(t, dir, avoirProto, "rpc DeleteAvoir")
				})
			}
			if s.shape == "aggregate" {
				assertFileExists(t, dir, mod+"/internal/application/command/facture_delete.go")
				assertFileNotExists(t, dir, mod+"/internal/application/command/facture_send.go")
				assertFileContains(t, dir, mod+"/internal/domain/facture/facture.go", "func (a *Facture) Delete(userID uuid.UUID)")
				assertFileExists(t, dir, mod+"/internal/adapters/outbound/events/outbox_dispatcher.go")
				assertFileNotExists(t, dir, mod+"/internal/adapters/outbound/gateway/facture_gateway.go")
				assertFileContains(t, dir, mod+"/internal/adapters/outbound/events/facture_events.go", "event.(*facture.Deleted)")
			}
			if s.shape == "gateway" {
				assertFileExists(t, dir, mod+"/internal/application/command/"+s.entity+"_send.go")
				assertFileExists(t, dir, mod+"/internal/application/command/"+s.entity+"_send_test.go")
				assertFileExists(t, dir, mod+"/internal/application/ports/"+s.entity+"_gateway.go")
				assertFileExists(t, dir, mod+"/internal/application/ports/publisher.go")
				assertFileNotExists(t, dir, mod+"/internal/application/ports/"+s.entity+"_repository.go")
				assertFileNotExists(t, dir, mod+"/internal/application/query/"+s.entity+".go")
				assertFileNotExists(t, dir, mod+"/internal/domain/"+s.entity+"/snapshot.go")
				assertFileContains(t, dir, mod+"/internal/domain/"+s.entity+"/"+s.entity+".go", "func (a *"+pascal+") MarkSent(userID uuid.UUID)")
				assertFileContains(t, dir, mod+"/internal/domain/"+s.entity+"/events.go", `EventTypeSent = "`+s.entity+`.sent"`)
				assertFileContains(t, dir, mod+"/internal/domain/"+s.entity+"/errors.go", "ErrGatewayUnavailable")
				assertFileContains(t, dir, mod+"/internal/application/"+s.entity+".go", "Send"+pascal+"(ctx context.Context, label string) error")
				assertFileContains(t, dir, mod+"/internal/testhelpers/"+s.entity+"_fake.go", "type Fake"+pascal+"Gateway struct")
				assertFileContains(t, dir, mod+"/internal/testhelpers/fakes.go", "type RecordingEventPublisher struct")
				assertFileNotExists(t, dir, mod+"/internal/adapters/outbound/persistence/postgres/"+s.entity+"_repository.go")
				assertFileExists(t, dir, mod+"/internal/adapters/outbound/gateway/"+s.entity+"_gateway.go")
				assertFileExists(t, dir, mod+"/internal/adapters/outbound/gateway/"+s.entity+"_gateway_test.go")
				assertFileExists(t, dir, mod+"/internal/adapters/outbound/events/bus_publisher.go")
				assertFileExists(t, dir, mod+"/internal/adapters/outbound/events/bus_publisher_test.go")
				events := mod + "/internal/adapters/outbound/events/" + s.entity + "_events.go"
				assertFileContains(t, dir, events, "event.(*"+s.entity+".Sent)")
				assertFileContains(t, dir, events, "= "+s.entity+".EventTypeSent")
				assertFileContains(t, dir, events, "= "+s.name+"def.Topic"+pascal+"Sent")
			}
			proto := "contracts/proto/" + s.name + "/v1/" + s.entity + ".proto"
			if s.shape == "aggregate" {
				assertFileContains(t, dir, proto, "rpc GetFacture(")
			}
			if s.shape == "gateway" {
				assertFileContains(t, dir, proto, "rpc Send"+pascal+"(Send"+pascal+"Request) returns (Send"+pascal+"Response);")
				assertFileContains(t, dir, proto, `option (options.v1.topic) = "`+s.name+"."+s.entity+`.sent.v1";`)
				assertFileContains(t, dir, "contracts/proto/"+s.name+"/v1/errors.proto", strings.ToUpper(s.name)+"_ERROR_CODE_GATEWAY_UNAVAILABLE = 4;")
				assertFileNotContains(t, dir, proto, "rpc Get"+pascal)
				assertFileContains(t, dir, mod+"/internal/adapters/inbound/inproc/"+s.entity+"_contract_adapter.go", "func (a *"+pascal+"ContractAdapter) Send"+pascal+"(")
				assertFileContains(t, dir, mod+"/internal/adapters/inbound/mapper/"+s.entity+"_errors.go", s.entity+".ErrGatewayUnavailable")
				assertFileNotExists(t, dir, mod+"/internal/adapters/inbound/mapper/"+s.entity+".go")
				assertFileContains(t, dir, mod+"/test/contract/"+s.entity+"_test.go", "ErrorCodeGatewayUnavailable")
				wiring := mod + "/" + s.entity + "_wiring.go"
				assertFileContains(t, dir, wiring, "gateway.NewHTTP"+pascal+"Gateway(cfg.Gateway.URL, cfg.Gateway.Token)")
				assertFileContains(t, dir, wiring, "events.NewBusPublisher(infra.EventBus)")
			}
			if s.grpc {
				assertFileExists(t, dir, mod+"/internal/adapters/inbound/connect/"+s.entity+"_handler.go")
				assertFileExists(t, dir, mod+"/internal/adapters/inbound/connect/auth_middleware.go")
			} else {
				assertFileNotExists(t, dir, mod+"/internal/adapters/inbound/connect")
			}
			root := mod + "/" + s.name + ".go"
			assertFileNotContains(t, dir, root, "Service()")
			assertFileContains(t, dir, mod+"/"+s.entity+"_wiring.go", "func (m *Module) "+pascal+"Service() "+s.name+"def."+pascal+"Service")
			assertFileExists(t, dir, mod+"/internal/adapters/outbound/events/bus_publisher.go")
			assertFileExists(t, dir, mod+"/internal/application/ports/publisher.go")
			assertFileContains(t, dir, mod+"/internal/testhelpers/fakes.go", "type RecordingEventPublisher struct")
			assertFileContains(t, dir, mod+"/internal/infra/config/config.go", "Gateway *Gateway")
			assertFileContains(t, dir, mod+"/internal/infra/config/configs/default.toml", "[gateway]")
			assertFileNotContains(t, dir, mod+"/mise.toml", "GATEWAY_TOKEN")
			assertFileNotContains(t, dir, mod+"/mise.toml", "DB_PASSWORD")
			assertFileContains(t, dir, mod+"/internal/infra/config/configs/default.toml", "token = \"fake-token\"")
			assertFileContains(t, dir, mod+"/internal/infra/config/config.go", "//go:embed configs/*\n")
			assertFileNotContains(t, dir, mod+"/internal/infra/config/config.go", "var conf *Config")
			assertFileContains(t, dir, mod+"/internal/infra/config/config.go", "RequireSecret(")
			assertFileNotContains(t, dir, mod+"/internal/infra/config/config.go", `Password == ""`)
			if s.database {
				assertFileContains(t, dir, root, "func Migrate(")
				assertFileContains(t, dir, root, "DBPool     *pgxpool.Pool")
				assertFileContains(t, dir, root, "if m.relay != nil")
				assertFileExists(t, dir, mod+"/internal/application/ports/uow.go")
				assertFileExists(t, dir, mod+"/internal/adapters/outbound/events/outbox_dispatcher.go")
				assertFileContains(t, dir, mod+"/internal/testhelpers/fakes.go", "type PassthroughUoW struct")
			} else {
				assertFileNotContains(t, dir, root, "pgxpool")
				assertFileNotContains(t, dir, root, "Migrate")
				assertFileNotExists(t, dir, mod+"/internal/application/ports/uow.go")
				assertFileNotExists(t, dir, mod+"/internal/application/ports/events.go")
				assertFileNotExists(t, dir, mod+"/internal/adapters/outbound/events/outbox_dispatcher.go")
				assertFileNotContains(t, dir, mod+"/internal/testhelpers/fakes.go", "PassthroughUoW")
				assertFileNotContains(t, dir, mod+"/mise.toml", "boa:generate")
				assertFileNotContains(t, dir, mod+"/mise.toml", "DB_URL")
				assertFileNotExists(t, dir, mod+"/cmd/migration/migrate.go")
				assertFileNotExists(t, dir, mod+"/sqlc.yaml")
			}
			if s.grpc {
				assertFileExists(t, dir, mod+"/cmd/"+s.name+"/main.go")
				assertFileExists(t, dir, mod+"/.air.toml")
				assertFileContains(t, dir, root, "func newHTTPServer(")
				assertFileContains(t, dir, root, "AuthSvc    defauth.AuthPrivateService")
			} else {
				assertFileNotExists(t, dir, mod+"/cmd/"+s.name)
				assertFileNotExists(t, dir, mod+"/.air.toml")
				assertFileNotContains(t, dir, root, "newHTTPServer")
				assertFileNotContains(t, dir, root, "AuthSvc")
				assertFileNotContains(t, dir, root, "func (m *Module) Handler()")
				assertFileNotContains(t, dir, mod+"/mise.toml", `[tasks."run:dev"]`)
				assertFileNotContains(t, dir, mod+"/internal/infra/config/config.go", "AuthServer")
			}
			t.Run("snippets insert at every marker and stay gofmt-clean", func(t *testing.T) {
				snippets, err := scaffold.RenderSnippets(s.vars())
				require.NoError(t, err)
				for _, sn := range snippets {
					ok, err := scaffold.InsertBefore(filepath.Join(dir, mod, sn.Path), sn.Marker, sn.Text)
					require.NoError(t, err)
					assert.True(t, ok, "marker %s in %s", sn.Marker, sn.Path)
				}
				assertGofmt(t, dir)
			})
		})
	}
}

// TestRenderImportOrder renders modules whose name sorts before "auth",
// "common" and "contracts", with short and long entity names. The contracts
// imports that carry the module name and the module's own imports must sit
// in their own gofmt groups, and no non-test line may exceed the lll limit,
// or the rendered files fail the workspace lint.
func TestRenderImportOrder(t *testing.T) {
	for _, s := range []shape{
		{"aggregate", true, true, "achat", "bon"},
		{"aggregate", true, false, "achat", "reglement"},
		{"gateway", false, true, "achat", "reglement"},
		{"gateway", false, false, "achat", "bon"},
		{"aggregate", true, true, "achat", "immobilisation"},
		{"read-model", true, true, "achat", "immobilisation"},
		{"gateway", false, true, "achat", "immobilisation"},
	} {
		t.Run(s.String(), func(t *testing.T) {
			dir := renderWith(t, s.vars())
			wireSlice(t, dir, s.vars())
			assertNoLeftoverActions(t, dir)
			assertGofmt(t, dir)
			assertLineLength(t, dir, maxLineLength)
		})
	}
}

// wireSlice inserts the slice's snippets at the module's markers, as AddSlice
// does, without renaming the fixture's migration stamps.
func wireSlice(t *testing.T, dir string, vars map[string]any) {
	t.Helper()
	snippets, err := scaffold.RenderSnippets(vars)
	require.NoError(t, err)
	name, _ := vars["Name"].(string)
	for _, s := range snippets {
		ok, err := scaffold.InsertBefore(filepath.Join(dir, "modules", name, s.Path), s.Marker, s.Text)
		require.NoError(t, err, s.Path)
		require.True(t, ok, "%s has no marker %s", s.Path, s.Marker)
	}
}

// maxLineLength is the lll limit of the workspace's .golangci.yml.
const maxLineLength = 124

// assertLineLength fails on any rendered non-test .go line longer than
// limit: the entity name appears several times on some lines, so a long
// entity must not push them past the linter. Test files are skipped: the
// workspace lint does not apply lll to them.
func assertLineLength(t *testing.T, dir string, limit int) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		rel, _ := filepath.Rel(dir, path)
		for i, line := range strings.Split(string(content), "\n") {
			assert.LessOrEqual(t, len([]rune(line)), limit, "%s:%d is longer than %d characters: %s", rel, i+1, limit, line)
		}

		return nil
	})
	require.NoError(t, err)
}

// TestSliceDoesNotOverwriteModule renders each tree alone and checks that the
// slice writes no path the module already wrote.
func TestSliceDoesNotOverwriteModule(t *testing.T) {
	files := func(fsys fs.FS) map[string]bool {
		dir := t.TempDir()
		m, err := goplt.LoadManifest(fsys)
		require.NoError(t, err)
		require.NoError(t, goplt.NewGenerator().Generate(fsys, m, dir, fixtureVars()))
		set := map[string]bool{}
		require.NoError(t, filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel, _ := filepath.Rel(dir, path)
				set[rel] = true
			}

			return err
		}))

		return set
	}
	module, slice := files(scaffold.ModuleFS()), files(scaffold.SliceFS())

	t.Run("no slice path equals a module path", func(t *testing.T) {
		var both []string
		for rel := range slice {
			if module[rel] {
				both = append(both, rel)
			}
		}
		assert.Empty(t, both)
	})
}
