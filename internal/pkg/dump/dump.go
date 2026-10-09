package dump

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
)

// Dump selects the rows cfg describes from conn and writes them to w as a psql
// data script. The row count of every table goes to summary.
func Dump(ctx context.Context, conn *pgx.Conn, cfg *Config, w, summary io.Writer) error {
	return withSnapshot(ctx, conn, func() error {
		cat, err := loadCatalog(ctx, conn)
		if err != nil {
			return err
		}
		if err := cat.applyTransforms(cfg.Transform); err != nil {
			return err
		}
		rows, err := walk(ctx, conn, cat, cfg)
		if err != nil {
			return err
		}
		rows.summary(summary)

		bw := bufio.NewWriter(w)
		if err := writeHeader(bw, cfg.Source); err != nil {
			return err
		}
		if err := writeData(ctx, conn, bw, cat, rows); err != nil {
			return err
		}
		if err := bw.Flush(); err != nil {
			return fmt.Errorf("dump: write: %w", err)
		}

		return nil
	})
}

// Options are the inputs of Run.
type Options struct {
	URL    string    // connection string of the database to dump
	Config string    // path of the dump config file
	Output string    // path of the script; Stdout when empty
	Stdout io.Writer // receives the script when Output is empty
	Stderr io.Writer // receives the row count of every table
}

// Run dumps the database as the config file describes. With an output path it
// writes through a temporary file renamed on success, so a failed dump never
// leaves a truncated script or clobbers the previous one.
func Run(ctx context.Context, opts *Options) error {
	f, err := os.Open(opts.Config)
	if err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	cfg, err := ParseConfig(f)
	_ = f.Close()
	if err != nil {
		return err
	}
	cfg.Source = opts.Config

	conn, err := pgx.Connect(ctx, opts.URL)
	if err != nil {
		return fmt.Errorf("dump: connect: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	if opts.Output == "" {
		return Dump(ctx, conn, cfg, opts.Stdout, opts.Stderr)
	}

	// A hidden temporary file next to the output, so the rename is atomic and
	// a leftover does not look like a script.
	out, err := os.CreateTemp(filepath.Dir(opts.Output), "."+filepath.Base(opts.Output)+".*.tmp")
	if err != nil {
		return fmt.Errorf("dump: %w", err)
	}
	tmp := out.Name()
	if err := Dump(ctx, conn, cfg, out, opts.Stderr); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)

		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("dump: %w", err)
	}
	// CreateTemp makes the file 0600; a script is read by other tools.
	if err := os.Chmod(tmp, 0o644); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("dump: %w", err)
	}
	if err := os.Rename(tmp, opts.Output); err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("dump: %w", err)
	}

	return nil
}
