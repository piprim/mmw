package checks

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

type lintChecker struct {
	dir    string // working directory for golangci-lint; empty = current directory
	out    io.Writer
	errOut io.Writer
}

// NewLintChecker returns a Checker that validates Go code via golangci-lint.
// golangci-lint output is streamed directly to out and errOut without buffering
// or parsing — the caller sees exactly what golangci-lint produces.
// golangci-lint must be installed and on PATH.
// targets should be Go package patterns (e.g. "./...", "./internal/pkg/checks").
// When targets is empty it defaults to "./...".
func NewLintChecker(out, errOut io.Writer) Checker {
	return &lintChecker{out: out, errOut: errOut}
}

// NewLintCheckerAt is like NewLintChecker but runs golangci-lint with dir as
// the working directory. Use this when linting a module that is not the current
// working directory (e.g. workspace-wide lint iteration).
func NewLintCheckerAt(dir string, out, errOut io.Writer) Checker {
	return &lintChecker{dir: dir, out: out, errOut: errOut}
}

func (c *lintChecker) Name() string {
	return "lint"
}

// Check runs `golangci-lint run <targets...>` and streams its output directly
// to the configured writers. The result signals violations when golangci-lint
// exits non-zero; the actual diagnostic text is already in out/errOut.
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

	args := append([]string{"run"}, targets...)
	golangciCmd := exec.CommandContext(ctx, "golangci-lint", args...)
	golangciCmd.Dir = c.dir
	golangciCmd.Stdout = c.out
	golangciCmd.Stderr = c.errOut

	runErr := golangciCmd.Run()

	if runErr != nil && ctx.Err() != nil {
		return Result{}, ctx.Err()
	}

	if runErr != nil {
		return result.withFailed(), nil
	}

	return result, nil
}
