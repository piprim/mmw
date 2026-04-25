# mmw

Platform library and developer tooling for the MMW modular monolith.

The `mmw` module provides two things: a **runtime platform** (`pkg/platform`) that modules depend on at runtime, and a **CLI** (`cmd/mmw`) used during development.

---

## Packages

### `pkg/platform` — runtime platform

Everything a module needs to participate in the monolith.

#### Module lifecycle

`mmw core` library defines the contract that every module must implement:

```go
type Module interface {
    Start(ctx context.Context) error
}
```

Each module defines and exposes his own module:

```go
type Infrastructure struct {
	DBPool     *pgxpool.Pool
	EventBus   pfevents.SystemEventBus
	Subscriber message.Subscriber
	AuthSvc    defauth.AuthPrivateService
	Logger     *slog.Logger
}

func New(infra Infrastructure) (*Module, error) {
	// Load the config
	cfg, err := config.Load(context.Background(), "")
	// Handle err

	// newApplicationService builds the infrastructure adapters (repository, outbox dispatcher,
	// unit of work) and wires them into the TodoApplicationService.
	todoService := newApplicationService(infra)

	// newEventRouter creates the Watermill message router and registers all inbound event
	// handlers for the Todo module.
	router, err := newEventRouter(infra)
	// Handle err

	// newHTTPServer mounts the Connect RPC handler on an HTTP mux (automatically wrapped with
	// platform middlewares by the platform), inject an token validator, wraps with an error
	// logging interceptor handling domain errors,
	// then returns a pre-configured HTTPServer ready to be started.
	httpServer := newHTTPServer(cfg, infra, todoService)

	return &Module{
		// Outbox relay: polls todo.event every 2 s and forwards rows to the SystemEventBus.
		relay:   pfoutbox.NewEventsRelay(infra.DBPool, infra.EventBus, infra.Logger, relayTableName),
		server:  httpServer,
		router:  router,
		logger:  infra.Logger,
		service: todoService,
	}, nil
}

// Start implements the module contract with a blocking process.
func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("starting the app")

	// Package errgroup provides synchronization, error propagation, and Context
	// cancellation for groups of goroutines working on subtasks of a common task.
	g, gCtx := errgroup.WithContext(ctx)

	// Start the HTTP server
	g.Go(func() error {
		return m.server.Start(gCtx)
	})

	// Start the Outbox relay
	if m.relay != nil {
		g.Go(func() error {
			m.relay.Start(gCtx)

			return nil
		})
	}

	// Start the Watermill message router triggering Todo module handlers for inbound event handlers.
	g.Go(func() error {
		return m.router.Run(gCtx)
	})

	// Wait until the context is cancled or a goroutine returns an error or panics.
	err := g.Wait()

	return eris.Wrapf(err, "%s failure", ModuleName)
}
```

The `main.go` simplified:

```go
func main() {
	// signal.NotifyContext cancels ctx on SIGINT / SIGTERM, which propagates a
	// graceful-shutdown signal to every running module via platform.Run.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	var dbPool *pgxpool.Pool

	defer func() {
		if dbPool != nil {
			dbPool.Close()
		}
		cancel()
		os.Exit(exitCode)
	}()

	// initObservability loads the application config and creates the structured logger.
	// If config.ServerDebugEnabled is true, it also starts a pprof server on localhost:6060 in the background.
	// Both resources are derived from config, so they belong together.
	config, logger, err := initObservability(ctx)
	// Handle error

	dbPool, err = getDatabasePoolConnexion(ctx, logger, config.MainDatabase.URL())
	// Handle error

	// Creates the in-process Watermill GoChannel and wraps it in the
	// platform SystemEventBus interface.
	// rawBus is the concrete GoChannel used directly by modules that need a
	// message.Subscriber (e.g. the todo module's event router, the notifications
	// module). eventBus is the publishing interface passed to every module so they
	// can emit domain events without depending on the Watermill type.
	rawBus := getRawbus(logger)
	eventBus := pfevents.NewWatermillBus(rawBus)
	defer rawBus.Close()

	// initModules wires and returns all application modules in dependency order.
	modules, err := initModules(logger, dbPool, rawBus, eventBus)
	if err != nil {
		return
	}

	// platform.Run launches every module in its own goroutine via errgroup and
	// blocks until the context is cancelled or one module fails.
	logger.Info("Platform startup…")
	if err = platform.New(logger, modules).Run(ctx); err != nil {
		logError(logger, "platform error", err)
	}
}

// initModules wires and returns all application modules in dependency order.
//
// Ordering matters: auth must be initialised before todo because todo's Connect
// handler requires an AuthPrivateService to validate JWT tokens. Notifications
// subscribes to topics from both auth and todo, so it is initialised last.
func initModules(
	logger *slog.Logger,
	dbPool *pgxpool.Pool,
	rawBus *gochannel.GoChannel,
	eventBus pfevents.SystemEventBus,
) ([]pfcore.Module, error) {
	// 1. Auth — no inter-module dependencies.
	authModule, err := auth.New(auth.Infrastructure{
		DBPool:   dbPool,
		EventBus: eventBus,
		Logger:   logger.With("module", auth.ModuleName),
	})
	// Handle error

	// 2. Todo — depends on auth's private service to validate bearer tokens.
	todoModule, err := todo.New(todo.Infrastructure{
		DBPool:     dbPool,
		EventBus:   eventBus,
		Subscriber: rawBus,
		Logger:     logger.With("module", todo.ModuleName),
		AuthSvc:    authModule.PrivateService(),
	})
	// Handle error

	// 3. Notifications — subscribes to domain events from both auth and todo.
	//    The topic list is built by merging the two modules' exported topic slices.
	notifModule, err := notifications.New(notifications.Infrastructure{
		Subscriber:  rawBus,
		Logger:      logger.With("module", notifications.ModuleName),
		Topics:      append(tododef.Topics, authdef.Topics...),
		WithNotifer: true,
	})
	// Handle error

	return []pfcore.Module{todoModule, authModule, notifModule}, nil
}
```


```go
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

#### Middleware (`pkg/platform/middleware`)

All middleware follows the standard `func(http.Handler) http.Handler` shape, aliased as `Middleware`. The four built-in pieces are composed by `server.NewHTTPServer` in the order shown above; modules can also apply individual middlewares directly to specific routes.

**`LoggingMiddleware`**

Logs every request after it completes using structured `slog`. Generates or propagates an `X-Request-ID` header (UUID) and stores it in the request context so downstream handlers and the recovery middleware can correlate logs.

Log level is derived from the response status code:

| Status range | Log level |
|---|---|
| < 404 | `INFO` |
| 404 | `WARN` |
| ≥ 500 | `ERROR` |

When `logPayloads` is `true`, the raw request body is captured and appended to the log entry (capped at 10 000 bytes). Bodies are **not** captured for gRPC/Connect streams (`Content-Type: application/grpc*` or `application/connect*`) because reading a streaming body would block the handler.

```go
middleware.LoggingMiddleware(logger, logPayloads bool) Middleware
```

**`RecoveryMiddleware`**

Catches any `panic` that escapes a handler, wraps it in an `eris` error to capture the full stack trace, logs it at `ERROR` level with the correlated `request_id`, and writes a generic `500` JSON response. The stack trace is never forwarded to the client.

```go
middleware.RecoveryMiddleware(logger) Middleware
```

**`CORSMiddleware`**

Configures CORS using the `rs/cors` library with the allowed methods, headers, and exposed headers required by Connect RPC and gRPC-Web (sourced from `connectrpc.com/cors`). Allowed origins default to `cfg.Host`; the `AllowedOrigins` slice in `config.Server` overrides this.

```go
middleware.CORSMiddleware(cfg *config.Server) Middleware
// Preflight cache: 7200 s
```

**`BearerAuthMiddleware`**

Enforces bearer-token authentication on every route not in `excludedPaths` (all `/debug/` paths are also always exempt). On a valid token it injects the authenticated user's UUID into the request context via `authctx.WithUserID` so application handlers can read it without depending on HTTP concerns.

```go
type TokenValidator func(ctx context.Context, token string) (uuid.UUID, error)

middleware.BearerAuthMiddleware(validate TokenValidator, logger, excludedPaths []string) Middleware
```

`TokenValidator` is a plain function type — modules provide their own implementation, typically a closure over the auth module's private service:

```go
func NewTokenValidator(svc defauth.AuthPrivateService) pfmiddleware.TokenValidator {
    return func(ctx context.Context, token string) (uuid.UUID, error) {
        resp, err := svc.ValidateToken(ctx, &authv1.ValidateTokenRequest{Token: token})
        if err != nil {
            return uuid.Nil, err
        }
        return uuid.Parse(resp.GetUserId())
    }
}

// Applied per-route:
authMiddleware := pfmiddleware.BearerAuthMiddleware(NewTokenValidator(infra.AuthSvc), logger, nil)
mux.Handle(path, authMiddleware(handler))
```

On any failure (missing header, empty token, validation error) the middleware writes HTTP 401 with a Connect-compatible JSON body and does **not** call the next handler:

```json
{"code":"unauthenticated","message":"missing or invalid token"}
```

**`GetRequestID`**

Reads the request ID injected by `LoggingMiddleware` from any context downstream:

```go
reqID := middleware.GetRequestID(r.Context())
```

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

The transactional outbox pattern solves the dual-write problem: without it, a command handler that both saves domain state *and* publishes an event to a message bus risks leaving the two out of sync if a crash or network error occurs between the two writes.

**How it works:**

Instead of publishing directly to the bus, the command handler writes events into an outbox table *inside the same database transaction* as the domain state change. Publishing to the bus is then delegated to a background relay that polls the table. Because both writes share a transaction, they either both commit or both roll back — the domain state and the pending events are always consistent.

```
Command handler (inside a DB transaction)
  ├── UPDATE todo SET ...          ← domain state
  └── INSERT INTO todo.event ...  ← outbox row (same tx)

EventsRelay (background, every 2 s)
  ├── SELECT ... FOR UPDATE SKIP LOCKED  ← fetch unpublished rows, lock them
  ├── bus.Publish(eventType, payload)    ← forward to SystemEventBus
  └── UPDATE todo.event SET published_at = NOW() ← mark done, commit
```

**Writing to the outbox — `PostgresOutboxDispatcher`:**

The dispatcher is the write side. It receives `[]domain.DomainEvent` collected during a command, serialises each to JSON (or Protobuf JSON when a proto mapping exists), and inserts the batch into the outbox table via `pgx.Batch`. Because it uses the `UnitOfWork` executor, the inserts automatically join the ambient transaction if one is active:

```go
err = c.unitOfWork.WithTransaction(ctx, func(txCtx context.Context) error {
    if err := c.repository.Save(txCtx, todo); err != nil {  // domain write
        return err
    }
    return c.eventDispatcher.Dispatch(txCtx, todo.Events()) // outbox write — same tx
})
```

**Reading from the outbox — `EventsRelay`:**

The relay is the read side. It runs as a `core.Module` goroutine, ticking every 2 seconds:

1. Opens a transaction and selects up to 100 unpublished rows with `FOR UPDATE SKIP LOCKED` — this prevents two relay instances from processing the same row simultaneously (safe to run multiple replicas).
2. Publishes each event to the `SystemEventBus`. If publishing fails the transaction is rolled back, leaving the rows unlocked for the next tick.
3. Marks successfully published rows with `published_at = NOW()` and commits.

```go
relay := outbox.NewEventsRelay(pool, eventBus, logger, "todo.event")
// relay implements core.Module — pass it to platform.New alongside the HTTP server
```

**Guarantees and trade-offs:**

| Property | Detail |
|---|---|
| At-least-once delivery | A crash between `Publish` and the `UPDATE` causes the relay to retry the row on the next tick. Consumers must be idempotent. |
| Ordering | Rows are fetched `ORDER BY occurred_at ASC`, preserving intra-aggregate event order within a batch. |
| Latency | Up to 2 s between domain write and bus publication (configurable). |
| Back-pressure | The relay processes at most 100 rows per tick; a large backlog drains at 50 rows/s at the default interval. |

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

**`StructArgs`** — reflects a struct's `db`-tagged fields into a `map[string]any` for use as named query parameters with `pgx.NamedArgs`. This keeps SQL queries readable with `@param` placeholders and removes the need to maintain a parallel positional argument list whenever the struct changes.

Define a snapshot struct with `db` tags (one tag per column name):

```go
type TodoSnapshot struct {
    ID          uuid.UUID  `db:"id"`
    Title       string     `db:"title"`
    Description string     `db:"description"`
    Status      string     `db:"status"`
    DueDate     *time.Time `db:"due_date"`   // pointer → NULL when nil
    UserID      uuid.UUID  `db:"user_id"`
    // ...
}
```

Pass it to `StructArgs` and cast to `pgx.NamedArgs`:

```go
query := `INSERT INTO todo.todo (id, title, description, status, due_date, user_id)
          VALUES (@id, @title, @description, @status, @due_date, @user_id)`

_, err := exec.Exec(ctx, query, pgx.NamedArgs(pfdb.StructArgs(todo.Snapshot())))
```

Tag rules:

| Tag value | Behaviour |
|---|---|
| `db:"col_name"` | included as `{"col_name": value}` |
| `db:"-"` | skipped |
| `db:"col_name,omitempty"` | included — `omitempty` is parsed but ignored (the DB handles `NULL` natively) |
| *(no tag)* | skipped |

`StructArgs` dereferences pointer receivers, so `StructArgs(&snap)` and `StructArgs(snap)` are equivalent. It panics if the value (after dereferencing) is not a struct.

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

#### Connect interceptors (`pkg/platform/connect`)

**`NewErrorLoggingInterceptor`**

A `connect.UnaryInterceptorFunc` that logs every handler error after the call returns, without altering the response seen by the client.

```go
connect.WithInterceptors(pfconnect.NewErrorLoggingInterceptor(logger))
```

**Why it exists — the wrapping problem**

Connect RPC handlers are expected to return `*connect.Error` values (with a gRPC status code) so the framework can serialise the error correctly for the client. This means each handler converts application-layer errors before returning:

```go
func (h *TodoHandler) CreateTodo(ctx context.Context, req *connect.Request[...]) (..., error) {
    todo, err := h.service.CreateTodo(ctx, &appReq)
    if err != nil {
        return nil, connectErrorFrom(err) // wraps into *connect.Error
    }
    // ...
}
```

`connectErrorFrom` (module-side) maps a `platform.DomainError` to the appropriate Connect code and attaches a typed proto error detail for clients:

```
application error (DomainError{Code: NotFound, ...})
    └─► connectErrorFrom
            └─► *connect.Error{code: CodeNotFound, detail: commonv1.DomainError{...}}
```

The wrapping discards the `eris` stack trace: `*connect.Error` does not carry it. The interceptor unwraps the `*connect.Error` to recover the original cause and logs the full stack trace before the wrapped error propagates to the framework:

```go
var connectErr *connect.Error
if errors.As(err, &connectErr) {
    if cause := connectErr.Unwrap(); cause != nil {
        errToLog = cause // original eris error, stack trace intact
    }
}
logger.Error("handler error", "procedure", req.Spec().Procedure, "err", errToLog)
```

**Error flow summary**

```
Handler returns connectErrorFrom(err)
    │
    ├── Interceptor fires (post-call)
    │     ├── unwraps *connect.Error → recovers eris cause
    │     └── logs procedure + full stack trace at ERROR
    │
    └── Connect framework serialises *connect.Error → HTTP response
          (client sees gRPC code + proto detail, never the stack trace)
```

The interceptor is registered when building the Connect handler in `todo.go`:

```go
path, handler := todov1connect.NewTodoServiceHandler(
    connecthandler.NewTodoHandler(todoService),
    connect.WithInterceptors(pfconnect.NewErrorLoggingInterceptor(infra.Logger)),
)
```

#### Structured logging

```go
logger, err := slog.New(slog.HandlerText, config.LogLevel.SlogLevel())
// slog.HandlerText or slog.HandlerJSON; integrates with lmittmann/tint for coloured output
```

#### Configuration (`pkg/platform/config`)

Layered configuration loader that combines embedded TOML files with environment variables.

**Loading order** (later sources win):
1. `configs/default.toml` — required baseline
2. `configs/<APP_ENV>.toml` — optional environment-specific overrides (e.g. `configs/development.toml`)
3. Environment variables — highest priority, applied last

**Defining a module config:**

Embed the TOML files and implement the `Config` interface. `Base` provides a ready-made implementation that reads `APP_ENV` from the environment:

```go
//go:embed configs/*.toml
var configFS embed.FS

type Config struct {
    config.Base                            // provides GetAppEnv() from APP_ENV env var
    Server      *pfconfig.Server          `mapstructure:"server"`
    MainDatabase pfconfig.Database        `mapstructure:"main-database"`
    LogLevel    LogLevel                   `mapstructure:"log-level"`
}

func Load(ctx context.Context, envPrefix string) (*Config, error) {
    cfg := &Config{}
    err := config.NewContext(ctx, configFS, envPrefix).Fill(cfg)
    return cfg, err
}
```

**Struct tags:**

| Tag | Source | Example |
|---|---|---|
| `mapstructure:"<key>"` | TOML file (kebab-case) | `mapstructure:"main-database"` |
| `env:"<VAR>"` | Environment variable | `env:"DB_PASSWORD"` |
| `env:"<VAR>, required"` | Required env var (error if missing) | `env:"APP_ENV, required"` |

**Built-in config types:**

`Database` — assembles a `postgres://` URL from individual fields; the password is sourced exclusively from `DB_PASSWORD` env var and excluded from JSON serialisation:

```go
type Database struct {
    Scheme   string `mapstructure:"scheme"`
    User     string `mapstructure:"user"`
    Password string `env:"DB_PASSWORD" json:"-"`
    Host     string `mapstructure:"host"`
    Port     Port   `mapstructure:"port"`
    Name     string `mapstructure:"name"`
    SSLMode  string `mapstructure:"sslmode"`
}

dbURL := cfg.MainDatabase.URL() // → "postgres://user:pass@host:5432/dbname?sslmode=disable"
```

`Server` — configures the platform HTTP server; safe defaults are applied by `SetDefaults()` if TOML fields are absent:

```go
type Server struct {
    Scheme            string        `mapstructure:"scheme"`
    Host              string        `mapstructure:"host"`
    Port              Port          `mapstructure:"port"`
    ReadHeaderTimeout time.Duration `mapstructure:"read-header-timeout"` // default: 5s
    IdleTimeout       time.Duration `mapstructure:"idle-timeout"`        // default: 120s
    ShutdownTimeout   time.Duration `mapstructure:"shutdown-timeout"`    // default: 30s
    AllowedOrigins    []string      `mapstructure:"allowed-origins"`
    DebugEnabled      bool          `env:"SERVER_DEBUG_ENABLED" mapstructure:"debug-enabled"`
}
```

`Environment` — typed string enum with `IsDev()` / `IsValid()` helpers and `UnmarshalText` support; valid values are `development`, `staging`, `production`, `testing`.

**Testing:** swap the embedded FS for an in-memory `fstest.MapFS` to avoid touching the filesystem:

```go
mockFS := fstest.MapFS{
    "configs/default.toml": &fstest.MapFile{Data: []byte(`[server]\nport = 8080`)},
}
cfg := &MyConfig{}
err := config.NewContext(ctx, mockFS, "").Fill(cfg)
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

Provides the embedded template tree and workspace helpers. Template rendering is delegated to [`goplt`](./goplt/README.md).

```go
// Embedded templates (default) or an external directory:
fsys := scaffold.EmbeddedFS()
fsys  = os.DirFS("/path/to/my-templates")

// Load manifest and render with goplt:
m, err   := goplt.LoadManifest(fsys)
vars     := map[string]any{"Name": "payment", "OrgPrefix": "github.com/acme", ...}
err       = goplt.NewGenerator().Generate(fsys, m, repoRoot, vars)

// Update workspace files after generation:
err = scaffold.UpdateGoWork(repoRoot, "payment")
err = scaffold.UpdateMiseToml(repoRoot, "payment")
```

Template functions available in every file via `goplt.DefaultFuncMap()`:

| Function | Example |
|---|---|
| `pascal` | `{{.Name \| pascal}}` → `Payment` |
| `lower` | `{{.Name \| lower}}` → `payment` |
| `snake` | `{{.Name \| snake}}` → `my_payment` |
| `camel` | `{{.Name \| camel}}` → `myPayment` |
| `kebab` | `{{.Name \| kebab}}` → `my-payment` |

**`template.toml` format:**

Variables can be declared as plain values (short form) or as a sub-table with an optional `description` shown in the TUI:

```toml
# short form
[variables]
name = ""

# sub-table form — description shown as subtitle in the interactive form
[variables.name]
default     = ""
description = "Module name in kebab-case (e.g. payment, order-management)"

[variables.org-prefix]
default     = "github.com/acme"
description = "Go module path prefix for your organisation"

[variables.contracts-path]
default     = ""
description = "Go module path of the shared contracts module"

[variables.platform-path]
default     = "github.com/piprim/mmw"
description = "Go module path of the MMW platform (rarely changed)"

[variables.with-connect]
default     = true
description = "Generate a Connect/gRPC inbound adapter and proto definition"

[variables.with-contract]
default     = true
description = "Generate the Go contract package (service interface, DTOs, errors)"

[variables.with-database]
default     = true
description = "Generate the PostgreSQL persistence adapter and migration scaffolding"

[variables.license]
default     = ["MIT", "BSD-3", "GNU GPL v3.0", "Apache Software License 2.0"]
description = "License to include in the module"

[conditions]
"modules"                                                                     = "{{if .WithModule}}true{{end}}"
"modules/{{.Name}}/internal/adapters/inbound/connect"                        = "{{if .WithConnect}}true{{end}}"
"modules/{{.Name}}/internal/adapters/inbound/inproc"                         = "{{if .WithContract}}true{{end}}"
"modules/{{.Name}}/internal/infra/persistence/migrations"                    = "{{if .WithDatabase}}true{{end}}"
"modules/{{.Name}}/cmd/migration"                                             = "{{if .WithDatabase}}true{{end}}"
"contracts/go/application"                                                    = "{{if .WithContract}}true{{end}}"
"contracts/proto"                                                             = "{{if and .WithContract .WithConnect}}true{{end}}"
```

`WithModule` is a routing flag set programmatically (`true` for `mmw new module`, `false` for `mmw new contract`) — it is not a TUI variable and is never shown to the user.

Variable names normalise automatically: `with-connect`, `with_connect`, and `withConnect` all map to `.WithConnect` in templates.

---

## CLI — `mmw`

```
mmw new module [--template <path>]        Scaffold a new module interactively
mmw new contract <name>                   Generate a contract definition
mmw check arch                            Validate architectural boundaries
mmw check files [--fix] [files...]        Check files for whitespace / EOF / size
mmw check format [--fix] [files...]       Check Go formatting with gofumpt
mmw check toml [files...]                 Check TOML syntax
mmw check yaml [files...]                 Check YAML syntax with yamllint
mmw check lint [--workspace] [packages…]  Run golangci-lint
mmw check pre-commit [--modified]         Run all checks as a pre-commit gate
mmw test coverage [flags] [packages]      Print a test coverage table
mmw workspace tidy                        go mod tidy every module then go work sync
mmw workspace status                      Verify module checksums and sync workspace
mmw workspace sync <module-dir>           Pin a module's HEAD commit across dependents
mmw workspace sync --all                  Sync every workspace module in order
mmw version                               Print version, commit, and build time
```

### `mmw new module`

Runs an interactive TUI form (via `goplt/tui`) built dynamically from the manifest's `[variables]` section. Each variable's `description` is shown as a subtitle. The org-prefix default is pre-filled from the workspace's `contracts/go.mod` when detected.

Prompts for module name, org prefix, contracts path, and feature flags, then:

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

### `mmw check files [--fix] [files...]`

Validates each file for:

| Check | Auto-fixable |
|---|---|
| Trailing whitespace on any line | Yes (`--fix`) |
| Missing newline at end of file | Yes (`--fix`) |
| File size > 500 KB | No |

Defaults to all git-tracked files when no arguments are given. `--fix` rewrites files in-place; size violations are never auto-fixed.

### `mmw check format [--fix] [files...]`

Reports any `.go` file whose content differs from what gofumpt would produce. Uses gofumpt as a library — no subprocess. `--fix` rewrites files in-place.

Defaults to all tracked `*.go` files when no arguments are given.

### `mmw check toml [files...]`

Parses each `.toml` file using `go-toml/v2` and reports syntax errors. No subprocess. Defaults to all tracked `*.toml` files.

### `mmw check yaml [files...]`

Runs `yamllint -d relaxed` against each `.yaml`/`.yml` file. Requires `yamllint` on PATH. Defaults to all tracked `*.yaml`/`*.yml` files.

### `mmw check lint [--workspace] [packages...]`

Runs `golangci-lint run` against the specified Go packages. Linting runs at package level (not per-file) so package-scope linters fire correctly. Requires `golangci-lint` on PATH. Defaults to `./...`.

`--workspace` iterates every module declared in `go.work` and lints each in turn. Used by the root `mise run lint` task.

### `mmw check pre-commit [--modified] [--fail-fast]`

Read-only orchestrator that runs all checks in sequence against git-selected files:

```
1. files  — trailing whitespace, EOF newline, size > 500 KB
2. yaml   — YAML syntax (yamllint)
3. toml   — TOML syntax (go-toml/v2)
4. format — gofumpt formatting
5. lint   — golangci-lint (package-level, derived from changed .go files)
```

**File selection:**

| Mode | Files checked |
|---|---|
| default | staged files only (`git diff --cached --diff-filter=ACM`) |
| `--modified` | staged + modified tracked files (for manual runs) |

`--fail-fast` stops after the first checker that reports violations. All other checks still run by default.

This command never modifies files or the git index — use the individual `--fix` commands for that.

**Usage in modules** (via `go tool`):

```toml
# mise.toml
[tasks.pre-commit]
depends = ["arch:check", "buf:lint"]
description = "Pre-commit checks"
run = "go tool mmw check pre-commit --modified"
alias = "pc"
```

```
# go.mod
tool (
    github.com/piprim/mmw/cmd/mmw
)
```

### `mmw workspace tidy`

Runs `go mod tidy` in every module declared in `go.work` (in declaration order), then runs `go work sync` at the workspace root.

Use after adding or removing dependencies in any module to keep all `go.sum` files consistent:

```bash
mmw workspace tidy
```

### `mmw workspace status`

Runs `go mod verify` in every workspace module to check that the module cache matches the expected checksums. Prints `✓ <module>` or `✗ <module>` per module, then runs `go work sync` at the workspace root. Exits non-zero if any module fails verification.

```bash
mmw workspace status
```

### `mmw workspace sync [--all] [module-dir]`

Pins a module's current HEAD commit across all other workspace modules that depend on it.

For a single module:

1. Runs `git rev-parse HEAD` in the module directory to obtain the commit hash.
2. Reads the Go module path from the module's `go.mod`.
3. For every other module in `go.work` whose `go.mod` references that module path, runs `go get <module>@<commit>` followed by `go mod tidy`.
4. Runs `go work sync` at the workspace root.

```bash
mmw workspace sync modules/auth          # sync auth's HEAD into its dependents
mmw workspace sync --all                 # sync every module in declaration order
```

| Flag | Description |
|---|---|
| `--all` | Sync every module declared in `go.work` in declaration order; the positional argument is ignored |

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
| `-s`, `--short` | `false` | Pass `-short` to skip integration tests |
| `-r`, `--run` | | Filter test names by regex |
| `-t`, `--timeout` | | Set test timeout (e.g. `2m`) |
| `-m`, `--min` | `0` | Exit 1 if any package falls below this % |

### `mmw version`

Prints the version, commit hash, and commit timestamp:

```
v1.2.3 commit=abc1234 built=2026-04-22T10:00:00Z
```

On a dirty working tree the commit hash is suffixed with `*`. When no tag is present the version is `dev`.

The version string is injected at build time via `-ldflags`; commit and timestamp are read from the VCS info embedded by `go build` (Go 1.18+, no extra flags needed). Build and install via mise:

```bash
mise run build    # → ./bin/mmw
mise run install  # → $GOBIN/mmw
```

---

## LLM policy

This project is in part assisted by LLMs.
