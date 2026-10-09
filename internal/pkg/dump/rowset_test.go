package dump

import (
	"strings"
	"testing"
)

func TestRowSetSummary(t *testing.T) {
	rows := rowSet{}
	rows.add("public.a", []string{"(0,1)"})
	rows.add("public.b", []string{"(0,1)", "(0,2)"})
	rows.add("public.c", nil)

	var b strings.Builder
	rows.summary(&b)

	t.Run("largest table first, empty tables left out", func(t *testing.T) {
		want := "       2  public.b\n       1  public.a\n"
		if b.String() != want {
			t.Errorf("summary =\n%q\nwant\n%q", b.String(), want)
		}
	})
}
