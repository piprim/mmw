package checks

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// yamllintProblem matches one line of yamllint's `-f parsable` output:
//
//	path/to/file.yaml:12:81: [error] message (rule)
//
// The path group is greedy so that paths containing a colon still parse.
var yamllintProblem = regexp.MustCompile(`^(.*):(\d+):\d+: (.*)$`)

type yamlChecker struct{}

// NewYAMLChecker returns a Checker that validates YAML syntax via yamllint.
// yamllint must be installed and on PATH. Non-YAML files in targets are skipped.
func NewYAMLChecker() Checker {
	return &yamlChecker{}
}

func (*yamlChecker) Name() string {
	return "yaml"
}

// Check runs `yamllint -d relaxed` against each .yaml/.yml file and reports its
// error-level problems.
//
// Warnings are not violations. The relaxed profile deliberately demotes stylistic
// rules such as line-length to warnings, and yamllint itself exits 0 when only
// warnings remain; --no-warnings keeps this check aligned with that verdict.
//
// When targets is empty it defaults to all *.yaml/*.yml files under the working directory.
// Returns an error if yamllint is not found on PATH or if the context is cancelled.
func (c *yamlChecker) Check(ctx context.Context, targets []string) (Result, error) {
	files, err := resolveTargets(ctx, targets, ".yaml", ".yml")
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

	args := append([]string{"-d", "relaxed", "-f", "parsable", "--no-warnings"}, yamlFiles...)
	cmd := exec.CommandContext(ctx, "yamllint", args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// yamllint exits non-zero when violations are found; that's expected.
	// Propagate context cancellation as a real error.
	runErr := cmd.Run()
	if runErr != nil && ctx.Err() != nil {
		return Result{}, fmt.Errorf("yaml: %w", ctx.Err())
	}

	for line := range strings.SplitSeq(strings.TrimSpace(out.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		result.Violations = append(result.Violations, parseYamllintProblem(line))
	}

	return result, nil
}

// parseYamllintProblem splits one line of `-f parsable` output into a Violation.
// A line that does not match the expected shape is kept verbatim so that nothing
// yamllint reports is silently dropped.
func parseYamllintProblem(line string) Violation {
	match := yamllintProblem.FindStringSubmatch(line)
	if match == nil {
		return Violation{Message: line}
	}

	lineNum, err := strconv.Atoi(match[2])
	if err != nil {
		return Violation{Message: line}
	}

	return Violation{
		File:    match[1],
		Line:    lineNum,
		Message: match[3],
	}
}
