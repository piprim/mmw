package checks

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type yamlChecker struct{}

// NewYAMLChecker returns a Checker that validates YAML syntax via yamllint.
// yamllint must be installed and on PATH. Non-YAML files in targets are skipped.
func NewYAMLChecker() Checker {
	return &yamlChecker{}
}

func (c *yamlChecker) Name() string {
	return "yaml"
}

// Check runs `yamllint -d relaxed` against each .yaml/.yml file.
// When targets is empty it defaults to all *.yaml/*.yml files under the working directory.
// Returns an error if yamllint is not found on PATH or if the context is cancelled.
func (c *yamlChecker) Check(ctx context.Context, targets []string) (Result, error) {
	files, err := c.resolveTargets(ctx, targets)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		CheckerName: c.Name(),
		Violations:  []Violation{},
	}

	yamlFiles := FilterByExt(files, ".yaml", ".yml")
	if len(yamlFiles) == 0 {
		return result, nil
	}

	if _, err := exec.LookPath("yamllint"); err != nil {
		return Result{}, fmt.Errorf("checks: yamllint not found on PATH: %w", err)
	}

	args := append([]string{"-d", "relaxed"}, yamlFiles...)
	cmd := exec.CommandContext(ctx, "yamllint", args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// yamllint exits non-zero when violations are found; that's expected.
	// Propagate context cancellation as a real error.
	runErr := cmd.Run()
	if runErr != nil && ctx.Err() != nil {
		return Result{}, ctx.Err()
	}

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

	return result, nil
}

func (c *yamlChecker) resolveTargets(ctx context.Context, targets []string) ([]string, error) {
	if len(targets) > 0 {
		return targets, nil
	}

	all, err := TrackedFiles(ctx)
	if err != nil {
		return nil, err
	}

	return FilterByExt(all, ".yaml", ".yml"), nil
}
