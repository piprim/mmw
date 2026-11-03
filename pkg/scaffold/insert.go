package scaffold

import (
	"fmt"
	"go/format"
	"os"
	"regexp"
	"slices"
	"strings"
)

// Markers are the comments module files carry where every slice registers
// itself; a snippet is inserted on the line before, the marker stays.
const (
	MarkerField   = "// mmw:slice-field"
	MarkerWire    = "// mmw:slice-wire"
	MarkerHandler = "// mmw:slice-handler"
	MarkerTopic   = "// mmw:slice-topic"
	MarkerEvent   = "// mmw:slice-event"
	MarkerError   = "// mmw:slice-error"
)

// InsertBefore inserts snippet (one or more lines) before the first line of
// path containing marker, indented like that line. It returns false when the
// marker is absent, and true without writing when the snippet's first line is
// already in the file, so a re-run does not insert twice. A .go file is
// gofmt'ed after the insertion (a new struct field or map key re-aligns its
// neighbours); if the result does not parse, nothing is written.
func InsertBefore(path, marker, snippet string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("insert before %s: %w", marker, err)
	}

	lines := strings.Split(string(data), "\n")
	at := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, marker) })
	if at < 0 {
		return false, nil
	}

	snippetLines := strings.Split(snippet, "\n")
	first := strings.Join(strings.Fields(snippetLines[0]), " ")
	if slices.ContainsFunc(lines, func(l string) bool { return strings.Join(strings.Fields(l), " ") == first }) {
		return true, nil
	}

	indent := lines[at][:len(lines[at])-len(strings.TrimLeft(lines[at], " \t"))]
	for i, l := range snippetLines {
		snippetLines[i] = indent + l
	}
	out := []byte(strings.Join(slices.Insert(lines, at, snippetLines...), "\n"))

	if strings.HasSuffix(path, ".go") {
		formatted, err := format.Source(out)
		if err != nil {
			return false, fmt.Errorf("insert before %s in %s: %w", marker, path, err)
		}
		out = formatted
	}

	//nolint:gosec // G703: path is a module file the scaffold just rendered or found, not user input
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return false, fmt.Errorf("insert before %s: %w", marker, err)
	}

	return true, nil
}

// bungenTablesRe matches bungen's table list in mise.toml.
var bungenTablesRe = regexp.MustCompile(`-t '([^']*)'`)

// AppendBungenTable adds table to the -t '…' list of mise.toml, unless it is
// already there. It returns false when the file has no such list.
func AppendBungenTable(miseTomlPath, table string) (bool, error) {
	data, err := os.ReadFile(miseTomlPath)
	if err != nil {
		return false, fmt.Errorf("append bungen table: %w", err)
	}

	m := bungenTablesRe.FindSubmatchIndex(data)
	if m == nil {
		return false, nil
	}
	if slices.Contains(strings.Split(string(data[m[2]:m[3]]), ","), table) {
		return true, nil
	}

	out := slices.Concat(data[:m[3]], []byte(","+table), data[m[3]:])
	//nolint:gosec // G703: the module's own mise.toml, not user input
	if err := os.WriteFile(miseTomlPath, out, 0o600); err != nil {
		return false, fmt.Errorf("append bungen table: %w", err)
	}

	return true, nil
}
