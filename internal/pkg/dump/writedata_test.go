//go:build integration

package dump

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestWriteData(t *testing.T) {
	db, _ := newDB(t, fixtureSchema, fixtureData)
	cfg := mustConfig(t, "seeds:\n  - table: status\n    child_depth: 0\n")
	ctx := context.Background()

	var out bytes.Buffer
	err := withSnapshot(ctx, db, func() error {
		cat, err := loadCatalog(ctx, db)
		if err != nil {
			return err
		}
		rows, err := walk(ctx, db, cat, cfg)
		if err != nil {
			return err
		}
		return writeData(ctx, db, &out, cat, rows)
	})
	if err != nil {
		t.Fatalf("writeData: %v", err)
	}

	t.Run("writes a COPY block per table, then the setvals and COMMIT", func(t *testing.T) {
		var want strings.Builder
		want.WriteString("COPY \"public\".\"status\" (\"id\", \"code\") FROM stdin;\n" +
			"1\tnew\n2\topen\n3\twon\n4\tlost\n" +
			"\\.\n\n" +
			"SELECT setval('public.status_id_seq', GREATEST(1, (SELECT max(\"id\") FROM ONLY \"public\".\"status\")));\n" +
			"\nCOMMIT;\n")
		if out.String() != want.String() {
			t.Errorf("script =\n%s\nwant\n%s", out.String(), want.String())
		}
	})
}

func TestDumpTransform(t *testing.T) {
	db, _ := newDB(t, fixtureSchema, fixtureData)
	ctx := context.Background()

	t.Run("a transformed column is written as the expression returns it", func(t *testing.T) {
		cfg := mustConfig(t, "seeds:\n  - table: status\n    child_depth: 0\ntransform:\n  status:\n    code: upper(t.code)\n")
		var out bytes.Buffer
		if err := Dump(ctx, db, cfg, &out, io.Discard); err != nil {
			t.Fatalf("Dump: %v", err)
		}
		if !strings.Contains(out.String(), "1\tNEW\n2\tOPEN\n3\tWON\n4\tLOST\n") {
			t.Errorf("script =\n%s\nwant the codes in upper case", out.String())
		}
	})

	t.Run("an expression that does not fit the column fails the dump", func(t *testing.T) {
		cfg := mustConfig(t, "seeds:\n  - table: a\n    child_depth: 0\ntransform:\n  a:\n    ts: \"'not a time'\"\n")
		err := Dump(ctx, db, cfg, io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "dump: copy public.a (transformed: ts)") {
			t.Fatalf("err = %v, want a copy error naming the transformed column", err)
		}
	})

	t.Run("an unknown transform column fails before any query", func(t *testing.T) {
		cfg := mustConfig(t, "seeds:\n  - table: a\ntransform:\n  a:\n    nope: \"1\"\n")
		err := Dump(ctx, db, cfg, io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "transform: column public.a.nope not found") {
			t.Fatalf("err = %v, want transform: column public.a.nope not found", err)
		}
	})
}
