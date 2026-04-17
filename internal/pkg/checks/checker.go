// Package checks provides pre-commit and CI check logic for the MMW platform.
// Each check implements the Checker interface; checks that support in-place
// fixing also implement Fixer.  The pre-commit orchestrator in precommit.go
// drives all checks without knowing their concrete types.
package checks

import "context"

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
}

// HasViolations reports whether the result contains at least one violation.
func (r Result) HasViolations() bool {
	return len(r.Violations) > 0
}

// Checker is implemented by every check.
// targets is the list of files or packages to inspect; each checker defines
// what an empty slice means (typically "all files of the relevant type").
type Checker interface {
	Name() string
	Check(ctx context.Context, targets []string) (Result, error)
}

// Fixer is optionally implemented by checkers that support --fix.
// Cobra commands type-assert Checker to Fixer at runtime; if the assertion
// fails the --fix flag is not registered for that command.
type Fixer interface {
	Checker
	Fix(ctx context.Context, targets []string) error
}
