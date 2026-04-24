package checks

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
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

func (*lintChecker) Name() string {
	return "lint"
}

// Check runs `golangci-lint run <targets...>` and streams its output directly
// to the configured writers. The result signals violations when golangci-lint
// exits non-zero; the actual diagnostic text is already in out/errOut.
//
// targets may be file paths (e.g. from a pre-commit file selection) or package
// patterns (e.g. "./..."). When file paths are given, package directories are
// derived from them automatically; if none of the files are Go files the check
// is skipped. When targets is empty it defaults to "./...".
func (c *lintChecker) Check(ctx context.Context, targets []string) (Result, error) {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		return Result{}, fmt.Errorf("checks: golangci-lint not found on PATH: %w", err)
	}

	runTargets, skip := resolveLintTargets(targets)
	if skip {
		fmt.Fprintln(c.out, "[lint] skipped (no Go files in selection)")

		return Result{CheckerName: c.Name()}, nil
	}

	result := Result{
		CheckerName: c.Name(),
		Violations:  []Violation{},
	}

	args := append([]string{"run"}, runTargets...)
	golangciCmd := exec.CommandContext(ctx, "golangci-lint", args...)
	golangciCmd.Dir = c.dir
	golangciCmd.Stdout = c.out
	golangciCmd.Stderr = c.errOut

	runErr := golangciCmd.Run()

	if runErr != nil && ctx.Err() != nil {
		return Result{}, fmt.Errorf("lint: %w", ctx.Err())
	}

	if runErr != nil {
		return result.withFailed(), nil
	}

	return result, nil
}

// resolveLintTargets returns the package patterns to pass to golangci-lint.
//
// If any target is a .go file path, packages are derived via PackageDirsFromFiles;
// skip is true when the derivation yields nothing (no Go files in the selection).
// Otherwise targets are used as-is, defaulting to ./... when empty.
func resolveLintTargets(targets []string) (pkgs []string, skip bool) {
	if hasGoFileTargets(targets) {
		pkgs = PackageDirsFromFiles(targets)

		return pkgs, len(pkgs) == 0
	}

	if len(targets) == 0 {
		return []string{"./..."}, false
	}

	return targets, false
}

// hasGoFileTargets reports whether any entry in targets has a .go extension,
// indicating that targets is a list of file paths rather than package patterns.
func hasGoFileTargets(targets []string) bool {
	for _, t := range targets {
		if filepath.Ext(t) == goExt {
			return true
		}
	}

	return false
}
