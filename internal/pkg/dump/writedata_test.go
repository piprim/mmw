//go:build integration

package dump

import (
	"bytes"
	"context"
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
