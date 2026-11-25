package enum

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "enums.toml")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("write config = %v", err)
	}
	return file
}

func TestLoadConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := LoadConfig(writeConfig(t, `
output = "internal/domain/ref"

[[enum]]
table = "public.evenement_type"
id    = "id"
label = "label"

[[enum]]
table = "core.event"
id    = "id"
label = "name"
`))
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		want := Config{
			Output: "internal/domain/ref",
			Enums: []Entry{
				{Table: "public.evenement_type", ID: "id", Label: "label"},
				{Table: "core.event", ID: "id", Label: "name"},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("LoadConfig() = %+v, want %+v", got, want)
		}
	})

	t.Run("optional comment column", func(t *testing.T) {
		got, err := LoadConfig(writeConfig(t, "output = \"ref\"\n[[enum]]\ntable = \"evenement_type\"\nid = \"id\"\nlabel = \"sid\"\ncomment = \"label\"\n"))
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if got.Enums[0].Comment != "label" {
			t.Errorf("Comment = %q, want %q", got.Enums[0].Comment, "label")
		}
	})

	t.Run("optional allow_zero", func(t *testing.T) {
		got, err := LoadConfig(writeConfig(t, "output = \"ref\"\n[[enum]]\ntable = \"vente_etape\"\nid = \"id\"\nlabel = \"label\"\nallow_zero = true\n"))
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if !got.Enums[0].AllowZero {
			t.Error("AllowZero = false, want true")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "nope.toml")
		_, err := LoadConfig(file)
		if err == nil || !strings.Contains(err.Error(), file) {
			t.Errorf("LoadConfig() error = %v, want it to name %s", err, file)
		}
	})

	const entry = "[[enum]]\ntable = \"public.evenement_type\"\nid = \"id\"\nlabel = \"label\"\n"

	tests := []struct {
		name     string
		toml     string
		contains []string
	}{
		{"no output", entry, []string{"output"}},
		{"empty output", "output = \"\"\n" + entry, []string{"output"}},
		{"no enum entry", "output = \"ref\"\n", []string{"no [[enum]]"}},
		{"entry without table", "output = \"ref\"\n[[enum]]\nid = \"id\"\nlabel = \"label\"\n", []string{"table"}},
		{"table with an empty name", "output = \"ref\"\n[[enum]]\ntable = \"public.\"\nid = \"id\"\nlabel = \"label\"\n", []string{"public.", "not a valid table"}},
		{"table with an empty schema", "output = \"ref\"\n[[enum]]\ntable = \".status\"\nid = \"id\"\nlabel = \"label\"\n", []string{".status", "not a valid table"}},
		{"table named like a Go keyword", "output = \"ref\"\n[[enum]]\ntable = \"type\"\nid = \"id\"\nlabel = \"label\"\n", []string{"type", "not a valid table"}},
		{"table with three parts", "output = \"ref\"\n[[enum]]\ntable = \"a.b.c\"\nid = \"id\"\nlabel = \"label\"\n", []string{"a.b.c", "not a valid table"}},
		{"table starting with a digit", "output = \"ref\"\n[[enum]]\ntable = \"2fa_type\"\nid = \"id\"\nlabel = \"label\"\n", []string{"2fa_type", "not a valid table"}},
		{"entry without id", "output = \"ref\"\n[[enum]]\ntable = \"t_one\"\nlabel = \"label\"\n", []string{"id", "t_one"}},
		{"entry without label", "output = \"ref\"\n[[enum]]\ntable = \"t_one\"\nid = \"id\"\n", []string{"label", "t_one"}},
		{"unknown entry key", "output = \"ref\"\n" + entry + "lable = \"x\"\n", []string{"lable"}},
		{"unknown top-level key", "output = \"ref\"\nouput = \"x\"\n" + entry, []string{"ouput"}},
		{
			"same table name in two schemas",
			"output = \"ref\"\n[[enum]]\ntable = \"public.status\"\nid = \"id\"\nlabel = \"l\"\n[[enum]]\ntable = \"core.status\"\nid = \"id\"\nlabel = \"l\"\n",
			[]string{"status", "same package"},
		},
		{"same table twice", "output = \"ref\"\n" + entry + entry, []string{"evenementtype", "same package"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, tt.toml))
			if err == nil {
				t.Fatal("LoadConfig() error = nil, want an error")
			}
			for _, s := range tt.contains {
				if !strings.Contains(err.Error(), s) {
					t.Errorf("error %q does not contain %q", err, s)
				}
			}
		})
	}
}

func TestGenerate_InvalidConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		contains string
	}{
		{"no output", Config{Enums: []Entry{{Table: "a", ID: "id", Label: "l"}}}, "output is required"},
		{"no entry", Config{Output: "ref"}, "no [[enum]] entry"},
		{
			"two entries giving the same package",
			Config{Output: "ref", Enums: []Entry{
				{Table: "public.status", ID: "id", Label: "l"},
				{Table: "core.status", ID: "id", Label: "l"},
			}},
			"same package status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fails before connecting: the URL is never used.
			err := Generate(t.Context(), "", tt.cfg, nil)
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Errorf("Generate() error = %v, want it to contain %q", err, tt.contains)
			}
		})
	}
}
