package enum

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goEnumOutput = `const (
	// R1ID is a EvenementType of type R1_ID.
	// RDV d'estimation
	R1ID EvenementType = iota + 1
	// CongeID is a EvenementType of type Conge_ID.
	// Congé
	CongeID EvenementType = iota + 5
	// PersoID is a EvenementType of type Perso_ID.
	PersoID
	// CompanyEventID is a EvenementType of type Company_event_ID.
	// Événement d’entreprise, "interne"
	CompanyEventID EvenementType = iota + 14
)

// ErrInvalidEvenementType is returned for an unknown value.
// It wraps nothing.
var ErrInvalidEvenementType = errors.New("not a valid EvenementType")
`

const wantDoc = `const (
	// R1ID is a EvenementType of type "RDV d'estimation".
	R1ID EvenementType = iota + 1
	// CongeID is a EvenementType of type "Congé".
	CongeID EvenementType = iota + 5
	// PersoID is a EvenementType of type Perso_ID.
	PersoID
	// CompanyEventID is a EvenementType of type "Événement d’entreprise, "interne"".
	CompanyEventID EvenementType = iota + 14
)

// ErrInvalidEvenementType is returned for an unknown value.
// It wraps nothing.
var ErrInvalidEvenementType = errors.New("not a valid EvenementType")
`

func TestRewriteDoc(t *testing.T) {
	t.Run("the comment replaces the go-enum name", func(t *testing.T) {
		if got := string(RewriteDoc([]byte(goEnumOutput))); got != wantDoc {
			t.Errorf("RewriteDoc() =\n%s\nwant\n%s", got, wantDoc)
		}
	})

	t.Run("a second pass changes nothing", func(t *testing.T) {
		if got := string(RewriteDoc([]byte(wantDoc))); got != wantDoc {
			t.Errorf("RewriteDoc() changed an already rewritten file:\n%s", got)
		}
	})
}

func TestRewriteDocFiles(t *testing.T) {
	t.Run("rewrites the file in place", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "evenementtype_enum.go")
		if err := os.WriteFile(file, []byte(goEnumOutput), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RewriteDocFiles(file); err != nil {
			t.Fatalf("RewriteDocFiles() error = %v", err)
		}
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != wantDoc {
			t.Errorf("file =\n%s\nwant\n%s", got, wantDoc)
		}
	})

	t.Run("a missing file is an error naming it", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "nope_enum.go")
		err := RewriteDocFiles(file)
		if err == nil || !strings.Contains(err.Error(), file) {
			t.Errorf("RewriteDocFiles() error = %v, want it to name %s", err, file)
		}
	})
}
