package scaffold_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/piprim/mmw/pkg/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSlicesDeclareNoSharedName adds two slices of each shape to one module
// and fails when two files of the same package declare the same top-level
// name: a helper copied into every slice's test file breaks the second slice.
func TestSlicesDeclareNoSharedName(t *testing.T) {
	root := newModuleRoot(t) // facture, aggregate
	for _, s := range []struct{ entity, shape string }{
		{"devis", "aggregate"},
		{"avoir", "read-model"}, {"releve", "read-model"},
		{"relance", "gateway"}, {"rappel", "gateway"},
	} {
		_, err := scaffold.AddSlice(scaffold.SliceFS(), root, sliceVars(s.entity, s.shape))
		require.NoError(t, err)
	}

	// package key (dir + package clause) → name → first file declaring it
	seen := map[string]map[string]string{}
	var dups []string
	err := filepath.WalkDir(filepath.Join(root, "modules"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		pkg := filepath.Dir(rel) + ":" + f.Name.Name
		if seen[pkg] == nil {
			seen[pkg] = map[string]string{}
		}
		for _, name := range topLevelNames(f) {
			if first, ok := seen[pkg][name]; ok {
				dups = append(dups, name+" in "+rel+" and "+first)

				continue
			}
			seen[pkg][name] = rel
		}

		return nil
	})
	require.NoError(t, err)

	t.Run("no two files of a package declare the same top-level name", func(t *testing.T) {
		assert.Empty(t, dups)
	})
}

// topLevelNames lists the package-level funcs (not methods), types, vars and
// consts of f, skipping the blank identifier and init.
func topLevelNames(f *ast.File) []string {
	var names []string
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name != "init" {
				names = append(names, d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					names = append(names, s.Name.Name)
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if n.Name != "_" {
							names = append(names, n.Name)
						}
					}
				}
			}
		}
	}

	return names
}
