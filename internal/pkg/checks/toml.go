package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type tomlChecker struct{}

// NewTOMLChecker returns a Checker that validates TOML syntax using go-toml/v2.
// Non-.toml files in the target list are silently skipped.
func NewTOMLChecker() Checker {
	return &tomlChecker{}
}

func (c *tomlChecker) Name() string {
	return "toml"
}

// Check parses each .toml file in targets and reports syntax errors.
// When targets is empty it defaults to all *.toml files under the working directory.
func (c *tomlChecker) Check(ctx context.Context, targets []string) (Result, error) {
	files, err := c.resolveTargets(ctx, targets)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		CheckerName: c.Name(),
		Violations:  []Violation{},
	}

	for _, path := range files {
		if filepath.Ext(path) != ".toml" {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, fmt.Errorf("checks: toml: read %s: %w", path, err)
		}

		var v any
		if err := toml.Unmarshal(data, &v); err != nil {
			result.Violations = append(result.Violations, Violation{
				File:    path,
				Message: err.Error(),
			})
		}
	}

	return result, nil
}

func (c *tomlChecker) resolveTargets(ctx context.Context, targets []string) ([]string, error) {
	if len(targets) > 0 {
		return targets, nil
	}

	all, err := TrackedFiles(ctx)
	if err != nil {
		return nil, err
	}

	return FilterByExt(all, ".toml"), nil
}
