//go:build integration

package dump

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// envDSN points the integration tests at an existing server instead of a
// container. Its role must be allowed to create databases; the round trip
// also needs a superuser and skips otherwise.
const envDSN = "MMW_DUMP_TEST_DSN"

const (
	fixtureSchema = "testdata/dump_schema.sql"
	fixtureData   = "testdata/dump_data.sql"
)

// serverDSN is the server every test creates its own databases on.
var serverDSN string

func TestMain(m *testing.M) {
	if dsn := os.Getenv(envDSN); dsn != "" {
		serverDSN = dsn
		os.Exit(m.Run())
	}

	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintf(os.Stderr, "dump integration tests: docker not found; start Docker or set %s\n", envDSN)
		os.Exit(1)
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("dump"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dump integration tests: start postgres: %v\n", err)
		os.Exit(1)
	}
	serverDSN, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		fmt.Fprintf(os.Stderr, "dump integration tests: connection string: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	_ = testcontainers.TerminateContainer(container)
	os.Exit(code)
}

// newDB creates an empty database on the test server, loads files into it and
// returns a connection to it with its URL. The database is dropped when t ends.
func newDB(t *testing.T, files ...string) (*pgx.Conn, string) {
	t.Helper()
	ctx := context.Background()
	name := fmt.Sprintf("dump_%d", time.Now().UnixNano())

	admin, err := pgx.Connect(ctx, serverDSN)
	if err != nil {
		t.Fatalf("connect to the test server: %v", err)
	}
	_, err = admin.Exec(ctx, "CREATE DATABASE "+name)
	_ = admin.Close(ctx)
	if err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		admin, err := pgx.Connect(context.Background(), serverDSN)
		if err != nil {
			t.Logf("drop test database %s: %v", name, err)
			return
		}
		defer func() { _ = admin.Close(context.Background()) }()
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Logf("drop test database %s: %v", name, err)
		}
	})

	u, err := url.Parse(serverDSN)
	if err != nil {
		t.Fatalf("parse test DSN: %v", err)
	}
	u.Path = "/" + name
	dsn := u.String()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to %s: %v", name, err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	for _, file := range files {
		script, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(ctx, string(script)); err != nil {
			t.Fatalf("load %s: %v", file, err)
		}
	}
	return conn, dsn
}

func readCatalog(t *testing.T, conn *pgx.Conn) *catalog {
	t.Helper()
	ctx := context.Background()
	var cat *catalog
	err := withSnapshot(ctx, conn, func() error {
		var err error
		cat, err = loadCatalog(ctx, conn)
		return err
	})
	if err != nil {
		t.Fatalf("loadCatalog: %v", err)
	}
	return cat
}

func mustConfig(t *testing.T, text string) *Config {
	t.Helper()
	cfg, err := ParseConfig(strings.NewReader(text))
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return cfg
}
