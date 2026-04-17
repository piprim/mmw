package scaffold

import (
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Manifest holds the parsed content of a template.toml file.
type Manifest struct {
	// Variables is the list of template variables parsed from [variables].
	Variables []Variable
	// Conditions maps an unrendered path prefix to a Go template boolean expression.
	// A condition that evaluates to an empty string means "skip this subtree".
	Conditions map[string]string
}

// rawManifest is the TOML-decoded intermediate representation.
type rawManifest struct {
	Variables  map[string]any    `toml:"variables"`
	Conditions map[string]string `toml:"conditions"`
}

// LoadManifest reads and parses template.toml from fsys.
// Variable names are normalized to PascalCase (see NormalizeKey).
func LoadManifest(fsys fs.FS) (*Manifest, error) {
	data, err := fs.ReadFile(fsys, "template.toml")
	if err != nil {
		return nil, fmt.Errorf("read template.toml: %w", err)
	}

	var raw rawManifest
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse template.toml: %w", err)
	}

	m := &Manifest{
		Conditions: make(map[string]string, len(raw.Conditions)),
	}

	maps.Copy(m.Conditions, raw.Conditions)

	for rawName, val := range raw.Variables {
		v, err := parseVariable(rawName, val)
		if err != nil {
			return nil, err
		}

		m.Variables = append(m.Variables, v)
	}

	slices.SortFunc(m.Variables, func(a, b Variable) int {
		return strings.Compare(a.Name, b.Name)
	})

	return m, nil
}

func parseVariable(rawName string, val any) (Variable, error) {
	v := Variable{Name: NormalizeKey(rawName)}
	switch tv := val.(type) {
	case string:
		v.Kind = KindText
		v.Default = tv
	case bool:
		v.Kind = KindBool
		v.Default = tv
	case []any:
		choices := make([]string, len(tv))
		for i, c := range tv {
			s, ok := c.(string)
			if !ok {
				return Variable{}, fmt.Errorf("variable %q: choice values must be strings, got %T", rawName, c)
			}
			choices[i] = s
		}
		v.Kind = KindChoiceString
		v.Default = choices
	default:
		return Variable{}, fmt.Errorf("variable %q: unsupported type %T (use string, bool, or []string)", rawName, val)
	}

	return v, nil
}
