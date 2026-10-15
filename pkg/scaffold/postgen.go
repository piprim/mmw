package scaffold

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

// PostGenerate runs the commands a freshly rendered module needs, in order,
// and stops at the first failure. The files already written stay in place so
// the user can finish by hand.
//
//  1. mise run buf:generate (repository root): emits contracts/go/network/<name>
//     and contracts/go/application/<name>; needs protoc-gen-go-contracts and network.
//  2. go mod tidy (modules/<name>): fills the indirect requirements and go.sum.
//  3. go work sync (repository root).
func PostGenerate(ctx context.Context, out, errOut io.Writer, repoRoot, name string) error {
	moduleDir := filepath.Join(repoRoot, "modules", name)

	steps := []step{
		{repoRoot, "mise run buf:generate", []string{"mise", "run", "buf:generate"}},
		{moduleDir, "go mod tidy", []string{"go", "mod", "tidy"}},
		{repoRoot, "go work sync", []string{"go", "work", "sync"}},
	}

	return runSteps(ctx, out, errOut, steps)
}

type step struct {
	dir   string
	label string
	args  []string
}

// runSteps runs steps in order; on failure the error lists the steps left.
func runSteps(ctx context.Context, out, errOut io.Writer, steps []step) error {
	for i, s := range steps {
		_, _ = fmt.Fprintf(out, "→ %s (in %s)\n", s.label, s.dir)
		err := runStep(ctx, out, errOut, s.dir, s.label, s.args...)
		if err == nil {
			continue
		}

		var left strings.Builder
		for _, r := range steps[i+1:] {
			_, _ = fmt.Fprintf(&left, "\n  - %s (in %s)", strings.Join(r.args, " "), r.dir)
		}
		if left.Len() == 0 {
			return err
		}

		return fmt.Errorf("%w\nremaining steps to run by hand:%s", err, left.String())
	}

	return nil
}

// runStep runs one program in dir, streaming its output.
func runStep(ctx context.Context, out, errOut io.Writer, dir, label string, args ...string) error { //nolint:revive,lll // signature fixed
	//nolint:gosec // G204: args are fixed by PostGenerate, never user input
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = errOut

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"%s failed: %w (the generated files are kept; fix the cause and run the remaining steps by hand)",
			label, err,
		)
	}

	return nil
}
