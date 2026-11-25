package enum

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
)

// integerTypes are the postgres types an id column may have.
var integerTypes = map[string]bool{"int2": true, "int4": true, "int8": true}

// Options are the inputs of Run.
type Options struct {
	URL    string    // connection string of the database to read
	Config string    // path of the enum config file
	Stdout io.Writer // receives one line per generated file
}

// Run generates the enums the config file describes.
func Run(ctx context.Context, opts *Options) error {
	cfg, err := LoadConfig(opts.Config)
	if err != nil {
		return err
	}

	return Generate(ctx, opts.URL, cfg, opts.Stdout)
}

// Generate writes one go-enum source file per entry of cfg:
// <cfg.Output>/<pkg>/<pkg>.go. It only reads the database. Every file is built
// before any is written: if one table fails, nothing is written.
// summary receives one line per file and may be nil.
func Generate(ctx context.Context, url string, cfg Config, summary io.Writer) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("enum: %w", err)
	}
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("enum: connect: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	sources := make([][]byte, len(cfg.Enums))
	for i, e := range cfg.Enums {
		rows, err := read(ctx, conn, e)
		if err != nil {
			return err
		}
		if sources[i], err = Render(e, rows); err != nil {
			return err
		}
	}

	for i, e := range cfg.Enums {
		dir := filepath.Join(cfg.Output, e.Package())
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("%s: %w", e.Table, err)
		}
		file := filepath.Join(dir, e.Package()+".go")
		//nolint:gosec // G306: generated Go source, readable like any source file.
		if err := os.WriteFile(file, sources[i], 0o644); err != nil {
			return fmt.Errorf("%s: %w", e.Table, err)
		}
		if summary != nil {
			fmt.Fprintf(summary, "generated %s from %s\n", file, e.Table)
		}
	}

	return nil
}

// read checks the table and its two columns, then returns its rows ordered by id.
func read(ctx context.Context, conn *pgx.Conn, e Entry) ([]Row, error) {
	columns, err := conn.Query(ctx,
		"SELECT column_name, udt_name FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2",
		e.Schema(), e.Name())
	if err != nil {
		return nil, fmt.Errorf("%s: reading columns: %w", e.Table, err)
	}
	types := make(map[string]string)
	var name, udt string
	_, err = pgx.ForEachRow(columns, []any{&name, &udt}, func() error {
		types[name] = udt

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: reading columns: %w", e.Table, err)
	}
	if len(types) == 0 {
		return nil, fmt.Errorf("table %s.%s does not exist", e.Schema(), e.Name())
	}
	for _, column := range []string{e.ID, e.Label, e.Comment} {
		if _, ok := types[column]; !ok && column != "" {
			return nil, fmt.Errorf("%s: column %q does not exist", e.Table, column)
		}
	}
	if t := types[e.ID]; !integerTypes[t] {
		return nil, fmt.Errorf("%s: id column %q has type %s, want an integer (int2, int4, int8)", e.Table, e.ID, t)
	}

	// Names come from the config file: pgx.Identifier quotes them.
	comment := "NULL"
	if e.Comment != "" {
		comment = pgx.Identifier{e.Comment}.Sanitize()
	}
	query := fmt.Sprintf("SELECT %s::bigint, %s::text, %s::text FROM %s ORDER BY 1",
		pgx.Identifier{e.ID}.Sanitize(),
		pgx.Identifier{e.Label}.Sanitize(),
		comment,
		pgx.Identifier{e.Schema(), e.Name()}.Sanitize())
	result, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: reading rows: %w", e.Table, err)
	}
	rows, err := pgx.CollectRows(result, func(row pgx.CollectableRow) (Row, error) {
		var r Row
		if err := row.Scan(&r.ID, &r.Label, &r.Comment); err != nil {
			return r, fmt.Errorf("scan: %w", err)
		}

		return r, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: reading rows: %w", e.Table, err)
	}

	return rows, nil
}
