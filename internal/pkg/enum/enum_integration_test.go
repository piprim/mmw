//go:build integration

package enum

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// envDSN points the integration tests at an existing server instead of a
// container, like the dump tests. Its role must be allowed to create databases.
const envDSN = "MMW_DUMP_TEST_DSN"

// testDSN is the database loaded with testdata/enum.sql.
var testDSN string

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	ctx := context.Background()

	server := os.Getenv(envDSN)
	if server == "" {
		if _, err := exec.LookPath("docker"); err != nil {
			fmt.Fprintf(os.Stderr, "enum integration tests: docker not found; start Docker or set %s\n", envDSN)
			return 1
		}
		container, err := postgres.Run(ctx, "postgres:17-alpine",
			postgres.WithDatabase("enum"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "enum integration tests: start postgres: %v\n", err)
			return 1
		}
		defer func() { _ = testcontainers.TerminateContainer(container) }()
		if server, err = container.ConnectionString(ctx, "sslmode=disable"); err != nil {
			fmt.Fprintf(os.Stderr, "enum integration tests: connection string: %v\n", err)
			return 1
		}
	}

	drop, err := createDB(ctx, server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enum integration tests: %v\n", err)
		return 1
	}
	defer drop()

	return m.Run()
}

// createDB creates a database on server, loads the fixture into it and sets
// testDSN. The returned function drops the database.
func createDB(ctx context.Context, server string) (func(), error) {
	name := fmt.Sprintf("enum_%d", time.Now().UnixNano())

	admin, err := pgx.Connect(ctx, server)
	if err != nil {
		return nil, fmt.Errorf("connect to the test server: %w", err)
	}
	_, err = admin.Exec(ctx, "CREATE DATABASE "+name)
	_ = admin.Close(ctx)
	if err != nil {
		return nil, fmt.Errorf("create test database: %w", err)
	}
	drop := func() {
		admin, err := pgx.Connect(ctx, server)
		if err != nil {
			return
		}
		defer func() { _ = admin.Close(ctx) }()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
	}

	u, err := url.Parse(server)
	if err != nil {
		drop()
		return nil, fmt.Errorf("parse test DSN: %w", err)
	}
	u.Path = "/" + name
	testDSN = u.String()

	script, err := os.ReadFile("testdata/enum.sql")
	if err == nil {
		var conn *pgx.Conn
		if conn, err = pgx.Connect(ctx, testDSN); err == nil {
			_, err = conn.Exec(ctx, string(script))
			_ = conn.Close(ctx)
		}
	}
	if err != nil {
		drop()
		return nil, fmt.Errorf("load the fixture: %w", err)
	}

	return drop, nil
}

var (
	eventType = Entry{Table: "ref.event_type", ID: "id", Label: "label"}
	emptyType = Entry{Table: "ref.empty_type", ID: "id", Label: "label"}
)

func cfg(t *testing.T, entries ...Entry) Config {
	t.Helper()
	return Config{Output: t.TempDir(), Enums: entries}
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("file not generated = %v", err)
	}
	return string(content)
}

func TestGenerate(t *testing.T) {
	ctx := context.Background()

	t.Run("writes one package per table", func(t *testing.T) {
		golden := readFile(t, "testdata/eventtype.go.golden")
		c := cfg(t, eventType)
		var out strings.Builder
		if err := Generate(ctx, testDSN, c, &out); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		file := filepath.Join(c.Output, "eventtype", "eventtype.go")
		if generated := readFile(t, file); generated != golden {
			t.Errorf("generated does not match testdata/eventtype.go.golden:\n%s", generated)
		}
		if !strings.Contains(out.String(), file) {
			t.Errorf("summary %q does not name %s", out.String(), file)
		}
	})

	t.Run("label column is configurable", func(t *testing.T) {
		c := cfg(t, Entry{Table: "ref.event_type", ID: "id", Label: "sid"})
		if err := Generate(ctx, testDSN, c, io.Discard); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		generated := readFile(t, filepath.Join(c.Output, "eventtype", "eventtype.go"))
		for _, want := range []string{"r1_ID=1,", "company_event_ID=17,", "relance_r1_ID=28,"} {
			if !strings.Contains(generated, want) {
				t.Errorf("generated file does not contain %q:\n%s", want, generated)
			}
		}
	})

	t.Run("comment column is read", func(t *testing.T) {
		c := cfg(t, Entry{Table: "ref.event_type", ID: "id", Label: "sid", Comment: "label"})
		if err := Generate(ctx, testDSN, c, io.Discard); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		generated := readFile(t, filepath.Join(c.Output, "eventtype", "eventtype.go"))
		for _, want := range []string{"// r1_ID=1, // RDV d'estimation\n", "// company_event_ID=17, // Événement d’entreprise\n"} {
			if !strings.Contains(generated, want) {
				t.Errorf("generated file does not contain %q:\n%s", want, generated)
			}
		}
	})

	t.Run("a NULL comment in the database adds nothing", func(t *testing.T) {
		c := cfg(t, Entry{Table: "ref.null_label", ID: "id", Label: "id_name", Comment: "label"})
		if err := Generate(ctx, testDSN, c, io.Discard); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		generated := readFile(t, filepath.Join(c.Output, "nulllabel", "nulllabel.go"))
		if !strings.Contains(generated, "// trois_ID=3,\n") {
			t.Errorf("generated file does not contain the bare value:\n%s", generated)
		}
	})

	t.Run("a nil summary writer is allowed", func(t *testing.T) {
		c := Config{Output: t.TempDir(), Enums: []Entry{eventType}}
		if err := Generate(ctx, testDSN, c, nil); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
	})

	t.Run("a second run overwrites", func(t *testing.T) {
		c := cfg(t, eventType)
		if err := Generate(ctx, testDSN, c, io.Discard); err != nil {
			t.Fatalf("first Generate() error = %v", err)
		}
		c.Enums[0].Label = "sid"
		if err := Generate(ctx, testDSN, c, io.Discard); err != nil {
			t.Fatalf("second Generate() error = %v", err)
		}
		generated := readFile(t, filepath.Join(c.Output, "eventtype", "eventtype.go"))
		if !strings.Contains(generated, "r1_ID=1,") {
			t.Errorf("generated file does not contain the new names:\n%s", generated)
		}
		if strings.Contains(generated, "rdv_d_estimation_ID") {
			t.Errorf("generated file still contains a name of the first run:\n%s", generated)
		}
	})

	t.Run("relative output resolves against the working directory", func(t *testing.T) {
		tmp := t.TempDir()
		t.Chdir(tmp)
		if err := Generate(ctx, testDSN, Config{Output: "gen/ref", Enums: []Entry{eventType}}, io.Discard); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		readFile(t, filepath.Join(tmp, "gen", "ref", "eventtype", "eventtype.go"))
	})
}

func TestGenerate_Errors(t *testing.T) {
	tests := []struct {
		name     string
		entries  []Entry
		contains []string
	}{
		{"missing table", []Entry{{Table: "ref.nope", ID: "id", Label: "label"}}, []string{"ref.nope", "does not exist"}},
		{"missing id column", []Entry{{Table: "ref.event_type", ID: "nope", Label: "label"}}, []string{`"nope"`, "does not exist"}},
		{"missing label column", []Entry{{Table: "ref.event_type", ID: "id", Label: "nope"}}, []string{`"nope"`, "does not exist"}},
		{"missing comment column", []Entry{{Table: "ref.event_type", ID: "id", Label: "sid", Comment: "nope"}}, []string{`"nope"`, "does not exist"}},
		{"id column is not an integer", []Entry{{Table: "ref.text_id", ID: "id", Label: "label"}}, []string{"has type text", "want an integer"}},
		{"empty table", []Entry{emptyType}, []string{"no rows"}},
		{"NULL label in the database", []Entry{{Table: "ref.null_label", ID: "id", Label: "label"}}, []string{"id 3", "NULL label"}},
		{"a failing table keeps the valid one from being written", []Entry{eventType, emptyType}, []string{"no rows"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cfg(t, tt.entries...)
			err := Generate(context.Background(), testDSN, c, io.Discard)
			if err == nil {
				t.Fatal("Generate() error = nil, want an error")
			}
			for _, s := range tt.contains {
				if !strings.Contains(err.Error(), s) {
					t.Errorf("error %q does not contain %q", err, s)
				}
			}
			written, readErr := os.ReadDir(c.Output)
			if readErr != nil {
				t.Fatalf("read output dir = %v", readErr)
			}
			if len(written) != 0 {
				t.Errorf("output dir holds %d entries, want none", len(written))
			}
		})
	}
}
