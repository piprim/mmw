package checks

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// generatedMarker matches the comment Go tooling uses to mark machine-generated
// source. This is the rule implemented by go/ast.IsGenerated and applied by the
// `generated` exclusion in the repository .golangci.yml files: the comment must
// sit on its own line before the package clause.
var generatedMarker = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// IsGenerated reports whether path is a machine-generated Go source file.
//
// Only the header is read — scanning stops at the package clause or at the first
// line that is neither blank nor part of a comment — so the cost does not grow
// with file size.
func IsGenerated(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("checks: open %s: %w", path, err)
	}

	defer func() { _ = file.Close() }()

	generated, err := scanGeneratedMarker(file)
	if err != nil {
		return false, fmt.Errorf("checks: %s: %w", path, err)
	}

	return generated, nil
}

// scanGeneratedMarker reads the leading comment block of a Go source file and
// reports whether it carries the generated marker. Requiring the marker before
// the package clause is what keeps an occurrence in a string literal or a
// function body from being mistaken for a real header.
func scanGeneratedMarker(r io.Reader) (bool, error) {
	var inBlockComment bool

	scanner := newLargeBufScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))

		if inBlockComment {
			inBlockComment = !strings.Contains(line, "*/")

			continue
		}

		switch {
		case line == "":
			continue
		case generatedMarker.MatchString(line):
			return true, nil
		case strings.HasPrefix(line, "//"):
			continue
		case strings.HasPrefix(line, "/*"):
			inBlockComment = !strings.Contains(line, "*/")

			continue
		default:
			return false, nil // package clause or code: the header is over
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("scan: %w", err)
	}

	return false, nil
}
