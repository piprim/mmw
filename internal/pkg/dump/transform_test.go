package dump

import (
	"strings"
	"testing"
)

func transformCatalog() *catalog {
	status := &table{
		Schema: "public", Name: "status",
		Columns: []string{"id", "code"}, Types: []string{"integer", "text"}, PK: []string{"id"},
	}

	return &catalog{Tables: map[string]*table{"public.status": status}, Keys: []string{"public.status"}}
}

func TestApplyTransforms(t *testing.T) {
	t.Run("attaches the expressions to their table", func(t *testing.T) {
		cat := transformCatalog()
		if err := cat.applyTransforms(map[string]map[string]string{"status": {"code": "upper(t.code)"}}); err != nil {
			t.Fatal(err)
		}
		if got := cat.Tables["public.status"].Transforms["code"]; got != "upper(t.code)" {
			t.Errorf("transform = %q", got)
		}
	})

	t.Run("an unknown table is an error", func(t *testing.T) {
		err := transformCatalog().applyTransforms(map[string]map[string]string{"nope": {"code": "x"}})
		if err == nil || !strings.Contains(err.Error(), "transform: table public.nope not found") {
			t.Fatalf("err = %v, want transform: table public.nope not found", err)
		}
	})

	t.Run("an unknown column is an error", func(t *testing.T) {
		err := transformCatalog().applyTransforms(map[string]map[string]string{"status": {"nope": "x"}})
		if err == nil || !strings.Contains(err.Error(), "transform: column public.status.nope not found") {
			t.Fatalf("err = %v, want transform: column public.status.nope not found", err)
		}
	})
}

func TestCopySQLTransform(t *testing.T) {
	cat := transformCatalog()
	if err := cat.applyTransforms(map[string]map[string]string{"status": {"code": "upper(t.code)"}}); err != nil {
		t.Fatal(err)
	}

	t.Run("a transformed column is the expression cast to the column type", func(t *testing.T) {
		want := `COPY (SELECT t."id", (upper(t.code))::text AS "code" FROM ONLY "public"."status" t ` +
			`WHERE t.ctid = ANY('{"(0,1)"}'::tid[]) ORDER BY t."id") TO STDOUT`
		if got := copySQL(cat.Tables["public.status"], []string{"(0,1)"}); got != want {
			t.Errorf("copySQL =\n%s\nwant\n%s", got, want)
		}
	})
}
