# mmw

Platform library and developer tooling for the MMW modular monolith.

The `mmw` module provides two things: a **runtime platform** (`pkg/platform`) that modules depend on at runtime, and a **CLI** (`cmd/mmw-cli`) used during development.

---

## Packages

### `pkg/platform` — runtime platform

Everything a module needs to participate in the monolith.

#### Module lifecycle

```go
// core.Module is the contract every module implements.
type Module interface {
    Start(ctx context.Context) error
}

// platform.New wires modules together and runs them concurrently.
app := platform.New(logger, []core.Module{todoModule, authModule, notifModule})
err := app.Run(ctx) // blocks; cancels all modules when ctx is done or one fails
```

`App.Run` launches every module in its own goroutine via `errgroup`. A failure in any module cancels the shared context and returns the first error.

#### HTTP server

```go
srv := server.NewHTTPServer(server.HTTPServerInfra{
    Config:       &cfg.Server,
    Handler:      mux,           // your Connect / HTTP mux
    Logger:       logger,
    HealthFns:    map[string]func(context.Context) (any, error){"db": dbPing},
    ServiceNames: []string{"todo.v1.TodoService"}, // Needed by gRPC reflection
})
// srv implements core.Module — can be passed to platform.New
```

Built-in routes (always on):

| Route | Purpose |
|---|---|
| `GET /debug/monit` | Health / readiness probe (JSON) |

Routes available when `Config.DebugEnabled = true`:

| Route | Purpose |
|---|---|
| `GET /debug/info` | Build info (JSON) |
| `/debug/pprof/*` | Go pprof endpoints |
| gRPC reflection | `grpc.reflection.v1` + `v1alpha` to be used with [grpcui](https://github.com/fullstorydev/grpcui) |

Middleware chain (outside → in): **Logger → Recovery → CORS → Mux**. The whole chain is wrapped in `h2c` for HTTP/2 cleartext support (required by Connect RPC).

#### Events

```go
// SystemEventBus is the transport interface — in-memory, NATS, RabbitMQ, etc.
type SystemEventBus interface {
    Publish(ctx context.Context, eventType string, payload []byte) error
}

// Watermill adapter (GoChannel for in-process use):
rawGoChannel := gochannel.NewGoChannel(/* … */) //  <- subscriber
eventBus := events.NewWatermillBus(rawGoChannel) // <- publisher

// …

waterMillRouter.AddConsumerHandler(
	"todo.on_auth_user_deleted",
	"auth.user.deleted.v1",
	rawGoChannel,
	UserDeletedHandler,
)
```

#### Transactional outbox

```go
relay := outbox.NewEnventsRelay(pool, eventBus, logger, "todo_events")
// relay implements core.Module — polls the outbox table every 2 s and forwards events to the bus
```

#### PostgreSQL utilities

**Unit of Work** — abstracts `*pgxpool.Pool` and `pgx.Tx` behind one interface:

```go
type DBExecutor interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}
```

Usage:
1. `uow.Executor` returns a `DBExecutor`:
   ```go
   // Save persists a new todo to the database.
   func (r *PostgresTodoRepository) Save(ctx context.Context, todo *domain.Todo) error {
   	query := `INSERT INTO todo.todo (…) VALUES (…)`
   	_, err := r.uow.Executor(ctx).Exec(ctx, query, pgx.NamedArgs(/* … */))

   	return err
   }
   ```
2. `unitOfWork.WithTransaction`:
```go
// Execute Infrastructure operations within the Unit of Work so with transaction.
err = c.unitOfWork.WithTransaction(ctx, func(txCtx context.Context) error {
	// Use txCtx here so the repository uses the transaction if any!
	if err := c.repository.Save(txCtx, todo); err != nil {
		return eris.Wrap(err, "saving todo")
	}

	// Dispatch events using txCtx (e.g., saving to an Outbox table in the same DB)
	if err := c.eventDispatcher.Dispatch(txCtx, todo.Events()); err != nil {
		return eris.Wrap(err, "dispatching events")
	}

	return nil
})
```

**Database migrator** — wraps goose for structured migration execution.

#### Auth context

```go
// Inject a user ID into the request context (done by BearerAuthMiddleware):
ctx = authctx.WithUserID(ctx, userID)

// Read it anywhere downstream:
userID, ok := authctx.UserID(ctx)
```

`middleware.BearerAuthMiddleware` validates the `Authorization: Bearer <token>` header, calls a `TokenValidator` closure, and injects the UUID on success. Paths in `excludedPaths` and all `/debug/` routes are skipped.

usage:
```go
// NewTokenValidator wraps an AuthPrivateService as a platform TokenValidator.
// The returned function validates a bearer token by calling svc.ValidateToken
// and parses the user UUID from the response.
func NewTokenValidator(svc defauth.AuthPrivateService) pfmiddleware.TokenValidator {
	return func(ctx context.Context, token string) (uuid.UUID, error) {
		resp, err := svc.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
		if err != nil {
			//nolint:wrapcheck // err is not wrapped.
			return uuid.Nil, err
		}

		return uuid.Parse(resp.GetUserId())
	}
}

// …
authMiddleware := pfmiddleware.BearerAuthMiddleware(connecthandler.NewTokenValidator(infra.AuthSvc), infra.Logger, excludedPaths)
mux.Handle(path, authMiddleware(handler))
```

#### Structured logging

```go
logger, err := slog.New(slog.HandlerText, config.LogLevel.SlogLevel())
// slog.HandlerText or slog.HandlerJSON; integrates with lmittmann/tint for coloured output
```

---

### `pkg/archtest` — architectural boundary validation

Validates that modules respect the layered architecture rules defined for the monolith.

```go
exitCode := archtest.RunAll(repoRoot) // 0 = pass, 1 = fail
```

Built-in validators:

| Validator | Rule |
|---|---|
| `ContractPurityValidator` | Contract definitions must not import application or infrastructure packages |
| `LibDependencyValidator` | Shared libs (`libs/`) must not import module-specific code |
| `DomainPurityValidator` | Domain layer must not import adapters, infra, or application packages |
| `ApplicationPurityValidator` | Application layer must not import adapters or infra packages |

Per-module checks are also discovered and run via `mmw check arch` in each module directory (see the `mmw` cli).

---

### `pkg/scaffold` — cookiecutter-style module scaffolding

Generates new modules and contract definitions from a template tree driven by a `template.toml` manifest.

```go
// Use the embedded templates (default):
fsys := scaffold.EmbeddedFS()

// Or load from an external directory:
fsys = os.DirFS("/path/to/my-templates")

// Load manifest (reads template.toml from fsys):
m, err := scaffold.LoadManifest(fsys)

// Collect variables, enrich with derived values:
vars := map[string]any{"Name": "payment", "OrgPrefix": "github.com/acme", ...}
scaffold.EnrichVars(vars) // adds NameTitle, ModulePath, ContractsPath, PkgDef, PlatformPath

// Generate:
err = scaffold.GenerateModule(fsys, repoRoot, vars)
err = scaffold.GenerateContract(fsys, repoRoot, vars)
```

**`template.toml` format:**

```toml
[variables]
name          = ""         # text input (empty = required)
org-prefix    = "github.com/acme"  # text with default
with-connect  = true       # bool confirm
license       = ["MIT", "BSD-3"]   # select (first = default)

[conditions]
"modules/{{.Name}}/internal/adapters/inbound/connect" = "{{if .WithConnect}}true{{end}}"
```

Variable names normalise automatically: `with-connect`, `with_connect`, and `withConnect` all map to `.WithConnect` in templates.

---

## CLI — `mmw-cli`

```
mmw new module [--template <path>]   Scaffold a new module interactively
mmw new contract <name>              Generate a contract definition
mmw check arch                       Validate architectural boundaries
mmw test coverage [flags]            Print a test coverage table
```

### `mmw new module`

Runs an interactive `huh` form built dynamically from the manifest's `[variables]` section. Prompts for module name, org prefix, and feature flags, then:

1. Generates the full module tree under `modules/<name>/`
2. Generates contract definitions under `contracts/` (when `with-contract = true`)
3. Adds the new module to `go.work`
4. Registers the module in the root `mise.toml`

Pass `--template <path>` to use a custom template directory instead of the embedded defaults.

### `mmw new contract <name>`

Generates only the contract definition files for an existing module:
- `contracts/definitions/<name>/` — Go interface, DTOs, errors, in-process client
- `contracts/proto/<name>/v1/` — Protobuf service definition (when `with-connect = true`)

### `mmw check arch`

Runs all architectural boundary validators (see `pkg/archtest`) and exits non-zero on failure. Used as a pre-commit hook via `mise run arch:check`.

### `mmw test coverage`

Runs `go test -cover` on the current module and prints a formatted table:

```
┌─────────────────────────────────┬──────────┬──────────────┐
│ Package                         │ Coverage │ Status       │
├─────────────────────────────────┼──────────┼──────────────┤
│ pkg/scaffold                    │ 87.3%    │ Good         │
├─────────────────────────────────┼──────────┼──────────────┤
│ pkg/platform/server             │ 42.1%    │ Partial      │
└─────────────────────────────────┴──────────┴──────────────┘
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `-p`, `--packages` | `./...` | Package pattern |
| `-s`, `--short` | `false` | Pass `-short` to skip integration tests |
| `-r`, `--run` | | Filter test names by regex |
| `-t`, `--timeout` | | Set test timeout (e.g. `2m`) |
| `-m`, `--min` | `0` | Exit 1 if any package falls below this % |
