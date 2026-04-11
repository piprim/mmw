package scaffold

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// VariableKind represents the type of a template variable.
type VariableKind int

const (
	KindText   VariableKind = iota // string input (empty default = required)
	KindBool                       // confirm (yes/no)
	KindChoice                     // select from list (first item = default)
)

// Variable describes a single template variable loaded from template.toml.
type Variable struct {
	Name    string       // PascalCase (e.g. "WithConnect")
	Default any          // string | bool | []string
	Kind    VariableKind
}

// NormalizeKey converts a variable name from any supported format to PascalCase.
// Supports: hyphen-case (with-connect), snake_case (with_connect),
// camelCase (withConnect). All produce "WithConnect".
func NormalizeKey(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) <= 1 {
		// single word or camelCase — uppercase first rune only
		if s == "" {
			return s
		}
		r, size := utf8.DecodeRuneInString(s)
		return string(unicode.ToUpper(r)) + s[size:]
	}
	var b strings.Builder
	for _, p := range parts {
		if p != "" {
			r, size := utf8.DecodeRuneInString(p)
			b.WriteRune(unicode.ToUpper(r))
			b.WriteString(p[size:])
		}
	}
	return b.String()
}

// EnrichVars adds derived keys to the collected vars map.
// Must be called after user input is collected.
// Expects "Name" (string) and "OrgPrefix" (string) to be present.
func EnrichVars(vars map[string]any) error {
	name, ok := vars["Name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("scaffold: Name is required")
	}
	orgPrefix, _ := vars["OrgPrefix"].(string)
	if orgPrefix == "" {
		orgPrefix = "github.com/acme"
	}
	r, size := utf8.DecodeRuneInString(name)
	vars["NameTitle"] = string(unicode.ToUpper(r)) + name[size:]
	vars["ModulePath"] = orgPrefix + "/mmw-" + name
	vars["ContractsPath"] = orgPrefix + "/mmw-contracts"
	if _, ok := vars["PlatformPath"]; !ok {
		vars["PlatformPath"] = "github.com/piprim/mmw"
	}
	vars["PkgDef"] = "def" + name
	return nil
}
