package checks

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type lintChecker struct{}

// NewLintChecker returns a Checker that validates Go code via golangci-lint.
// golangci-lint must be installed and on PATH.
// targets should be Go package patterns (e.g. "./...", "./internal/pkg/checks").
// When targets is empty it defaults to "./...".
func NewLintChecker() Checker {
	return &lintChecker{}
}

func (c *lintChecker) Name() string {
	return "lint"
}

// Check runs `golangci-lint run <targets...>` and reports any output as violations.
// Linting runs at package level (not per-file) so all linters fire correctly,
// including package-scope linters like revive argument-limit.
func (c *lintChecker) Check(ctx context.Context, targets []string) (Result, error) {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		return Result{}, fmt.Errorf("checks: golangci-lint not found on PATH: %w", err)
	}

	if len(targets) == 0 {
		targets = []string{"./..."}
	}

	result := Result{
		CheckerName: c.Name(),
		Violations:  []Violation{},
	}

	args := append([]string{"run", "--out-format", "line-number"}, targets...)
	cmd := exec.CommandContext(ctx, "golangci-lint", args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if runErr := cmd.Run(); runErr != nil {
		// Propagate context cancellation as a real error.
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}

		// golangci-lint exits non-zero when violations are found.
		for line := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			result.Violations = append(result.Violations, Violation{
				File:    extractFileFromLine(line),
				Message: line,
			})
		}
	}

	return result, nil
}

// extractFileFromLine extracts the file path from a linter output line.
// Expected format: "path/file.go:10:5: message"
func extractFileFromLine(line string) string {
	before, _, found := strings.Cut(line, ":")
	if !found {
		return ""
	}

	return strings.TrimSpace(before)
}
