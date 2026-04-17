// coverage prints a formatted test coverage table for every Go package.
//
// Usage:
//
//	go run coverage.go [flags]
//
// Examples:
//
//	go run coverage.go
//	go run coverage.go --short
//	go run coverage.go --packages ./pkg/platform/...
//	go run coverage.go --run TestFoo --min 80
package test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const (
	fullpct    = 100.0
	goodpct    = 80.0
	partialpct = 50.0
	lowpct     = 20.0
)

type row struct {
	pkg      string
	cov      string
	pct      float64
	status   string
	hasTests bool
}

// moduleFromGoMod reads the module name from go.mod in the current directory.
// Using "go list -m" is unreliable in Go workspaces because it returns every module.
func moduleFromGoMod() (string, error) {
	f, err := os.Open("go.mod")
	if err != nil {
		return "", fmt.Errorf("open go.mod: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}

	return "", errors.New("module directive not found in go.mod")
}

func coverageStatus(pct float64) string {
	switch {
	case pct == fullpct:
		return "Full"
	case pct >= goodpct:
		return "Good"
	case pct >= partialpct:
		return "Partial"
	case pct >= lowpct:
		return "Low"
	default:
		return "Critical gap"
	}
}

// parse output line: ok  \t<pkg>\t<duration>\tcoverage: XX.X% of statements
func parseOKLine(line, module string) *row {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	pkg := strings.TrimPrefix(fields[1], module+"/")

	var pct float64
	for _, field := range fields {
		if before, ok := strings.CutSuffix(field, "%"); ok {
			if v, err := strconv.ParseFloat(before, 64); err == nil {
				pct = v
			}

			break
		}
	}

	return &row{
		pkg:      pkg,
		cov:      fmt.Sprintf("%.1f%%", pct),
		pct:      pct,
		status:   coverageStatus(pct),
		hasTests: true,
	}
}

func parseOutput(output, module string) []row {
	var rows []row
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "ok"):
			// ok  \t<pkg>\t<duration>\tcoverage: XX.X% of statements
			row := parseOKLine(line, module)
			if row == nil {
				continue
			}
			rows = append(rows, *row)

		case strings.HasPrefix(line, "?"):
			// ?   \t<pkg>\t[no test files]
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			pkg := strings.TrimPrefix(fields[1], module+"/")
			rows = append(rows, row{
				pkg:      pkg,
				cov:      "—",
				status:   "No test files",
				hasTests: false,
			})
		}
	}

	return rows
}

func printTable(w io.Writer, rows []row) {
	w1, w2, w3 := len("Package"), len("Coverage"), len("Status")
	for _, r := range rows {
		if len(r.pkg) > w1 {
			w1 = len(r.pkg)
		}
		if len(r.cov) > w2 {
			w2 = len(r.cov)
		}
		if len(r.status) > w3 {
			w3 = len(r.status)
		}
	}

	hline := func(left, mid, right string) {
		fill := "─"
		fmt.Fprint(w, left)
		fmt.Fprint(w, strings.Repeat(fill, w1+2))
		fmt.Fprint(w, mid)
		fmt.Fprint(w, strings.Repeat(fill, w2+2))
		fmt.Fprint(w, mid)
		fmt.Fprint(w, strings.Repeat(fill, w3+2))
		fmt.Fprintln(w, right)
	}
	dataRow := func(p, c, s string) {
		fmt.Fprintf(w, "│ %-*s │ %-*s │ %-*s │\n", w1, p, w2, c, w3, s)
	}

	hline("┌", "┬", "┐")
	dataRow("Package", "Coverage", "Status")
	hline("├", "┼", "┤")
	for i, r := range rows {
		dataRow(r.pkg, r.cov, r.status)
		if i < len(rows)-1 {
			hline("├", "┼", "┤")
		}
	}
	hline("└", "┴", "┘")
}

// --- Command ---

type options struct {
	tags []string
	// packages string
	short   bool
	run     string
	timeout string
	min     float64
}

func NewCoverageCmd() *cobra.Command {
	opts := &options{}

	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Print a test coverage table for all packages",
		Long: `coverage runs go test -cover and displays the results as a
formatted table with a status label derived from the coverage percentage.

Status thresholds:
  100%        → Full
  80–99%      → Good
  50–79%      → Partial
  20–49%      → Low
  0–19%       → Critical gap
  (no tests)  → No test files

Pass --min to enforce a minimum coverage threshold and exit with a
non-zero status if any package with tests falls below it.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, opts, args)
		},
	}

	f := cmd.Flags()
	// f.StringVarP(&opts.packages, "packages", "p", "./...", "package pattern passed to go test")
	f.BoolVarP(&opts.short, "short", "s", false, "pass -short to go test (skips integration tests)")
	f.StringVarP(&opts.run, "run", "r", "", "pass -run <regex> to go test (filter test names)")
	f.StringVarP(&opts.timeout, "timeout", "t", "", "pass -timeout <duration> to go test (e.g. 30s, 2m)")
	f.Float64VarP(&opts.min, "min", "m", 0, "exit 1 if any tested package is below this coverage %")
	f.StringSliceVarP(&opts.tags, "tags", "", []string{}, "a comma-separated list of additional test tags")

	return cmd
}

func run(cmd *cobra.Command, opts *options, args []string) error {
	module, err := moduleFromGoMod()
	if err != nil {
		return fmt.Errorf("could not determine module name: %w", err)
	}

	packages := args
	if len(args) == 0 {
		packages = []string{"./..."}
	}

	testArgs := []string{"test"}
	testArgs = append(testArgs, packages...)
	testArgs = append(testArgs, "-cover")

	if opts.short {
		testArgs = append(testArgs, "-short")
	}
	if opts.run != "" {
		testArgs = append(testArgs, "-run", opts.run)
	}
	if opts.timeout != "" {
		testArgs = append(testArgs, "-timeout", opts.timeout)
	}

	if len(opts.tags) > 0 {
		testArgs = append(testArgs, "-tags", strings.Join(opts.tags, ","))
	}

	// Exit code of "go test" is intentionally ignored: a failing test suite
	// should still display the coverage table.
	goCmd := exec.CommandContext(context.Background(), "go", testArgs...)
	goCmd.Stderr = cmd.ErrOrStderr()
	out, _ := goCmd.Output()

	rows := parseOutput(string(out), module)
	if len(rows) == 0 {
		return errors.New("no packages found — run this from the module root")
	}

	printTable(cmd.OutOrStdout(), rows)

	// Enforce minimum coverage threshold (useful in CI).
	if opts.min > 0 {
		var below []string
		for _, r := range rows {
			if r.hasTests && r.pct < opts.min {
				below = append(below, fmt.Sprintf("  %-30s %.1f%%", r.pkg, r.pct))
			}
		}
		if len(below) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "\ncoverage below %.0f%% threshold:\n%s\n",
				opts.min, strings.Join(below, "\n"))

			return fmt.Errorf("%.0f%% minimum coverage threshold not met", opts.min)
		}
	}

	return nil
}
