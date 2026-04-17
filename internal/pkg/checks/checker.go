// Package checks provides pre-commit and CI check logic for the MMW platform.
// Each check implements the Checker interface; checks that support in-place
// fixing also implement Fixer.  The pre-commit orchestrator in precommit.go
// drives all checks without knowing their concrete types.
package checks

import (
	"context"
	"strings"
)

// Violation describes a single check issue at a specific location.
type Violation struct {
	File    string
	Line    int // 0 when the violation is not line-specific
	Message string
}

// Result holds the outcome of a single checker run.
type Result struct {
	CheckerName string
	Violations  []Violation
	failed      bool // set when the checker flagged failure without structured violations
}

// HasViolations reports whether the result contains at least one violation
// or the checker flagged a general failure without structured violations.
func (r Result) HasViolations() bool {
	return len(r.Violations) > 0 || r.failed
}

// withFailed returns a copy of r with the failed flag set.
// Use this in checkers that stream output directly and cannot populate
// structured Violations.
func (r Result) withFailed() Result {
	r.failed = true

	return r
}

// Checker is implemented by every check.
// targets is the list of files or packages to inspect; each checker defines
// what an empty slice means (typically "all files of the relevant type").
type Checker interface {
	Name() string
	Check(ctx context.Context, targets []string) (Result, error)
}

// extractFileFromLine extracts the file path from a tool output line.
// Expected format: "path/file.go:10:5: message"
func extractFileFromLine(line string) string {
	before, _, found := strings.Cut(line, ":")
	if !found {
		return ""
	}

	return strings.TrimSpace(before)
}

// Fixer is optionally implemented by checkers that support --fix.
// Cobra commands type-assert Checker to Fixer at runtime; if the assertion
// fails the --fix flag is not registered for that command.
type Fixer interface {
	Checker
	Fix(ctx context.Context, targets []string) error
}
