package checks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mvdan.cc/gofumpt/format"
)

type formatChecker struct{}

// NewFormatChecker returns a Checker (and Fixer) that validates Go source
// formatting using gofumpt.  No subprocess is required — gofumpt is called
// as a library via format.Source.
func NewFormatChecker() Checker {
	return &formatChecker{}
}

func (*formatChecker) Name() string {
	return "format"
}

// Check reports any .go file whose content differs from what gofumpt would produce.
// Non-.go files in targets are silently skipped.
// When targets is empty it defaults to all *.go files under the working directory.
func (c *formatChecker) Check(ctx context.Context, targets []string) (Result, error) {
	files, err := resolveTargets(ctx, targets, goExt)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		CheckerName: c.Name(),
		Violations:  []Violation{},
	}

	for _, path := range files {
		if filepath.Ext(path) != goExt {
			continue
		}

		formatted, err := isFormatted(path)
		if err != nil {
			return Result{}, fmt.Errorf("checks: format: %w", err)
		}

		if !formatted {
			result.Violations = append(result.Violations, Violation{
				File:    path,
				Message: "not formatted by gofumpt; run: mmw check format --fix",
			})
		}
	}

	return result, nil
}

// Fix rewrites each .go file in-place with the output of gofumpt.
func (*formatChecker) Fix(ctx context.Context, targets []string) error {
	files, err := resolveTargets(ctx, targets, goExt)
	if err != nil {
		return err
	}

	for _, path := range files {
		if filepath.Ext(path) != goExt {
			continue
		}

		if err := formatFile(path); err != nil {
			return fmt.Errorf("checks: format fix %s: %w", path, err)
		}
	}

	return nil
}

func isFormatted(path string) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}

	out, err := format.Source(src, format.Options{})
	if err != nil {
		return false, fmt.Errorf("gofumpt %s: %w", path, err)
	}

	return bytes.Equal(src, out), nil
}

func formatFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	out, err := format.Source(src, format.Options{})
	if err != nil {
		return fmt.Errorf("gofumpt: %w", err)
	}

	//nolint:gosec // G703: git path, not user input
	if err := os.WriteFile(path, out, info.Mode().Perm()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
