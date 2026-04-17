package checks

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// StagedFiles returns files staged for commit (ACM diff filter only).
// This is the correct set for a real git hook.
func StagedFiles(ctx context.Context) ([]string, error) {
	return gitDiffFiles(ctx, "--cached")
}

// ModifiedFiles returns the deduped union of staged and modified tracked files.
// This mirrors what `git commit -a` would commit and is suitable for manual runs.
func ModifiedFiles(ctx context.Context) ([]string, error) {
	staged, err := gitDiffFiles(ctx, "--cached")
	if err != nil {
		return nil, err
	}

	modified, err := gitDiffFiles(ctx)
	if err != nil {
		return nil, err
	}

	merged := slices.Concat(staged, modified)
	slices.Sort(merged)

	return slices.Compact(merged), nil
}

// TrackedFiles returns all files currently tracked by git under the working directory.
func TrackedFiles(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, "git", "ls-files").Output()
	if err != nil {
		return nil, fmt.Errorf("checks: git ls-files: %w", err)
	}

	return splitLines(string(out)), nil
}

// SelectFiles returns the appropriate file list based on the modified flag.
// Pass modified=true to include unstaged-but-tracked changes (manual run mode).
// Pass modified=false for staged-only mode (real git hook mode).
func SelectFiles(ctx context.Context, modified bool) ([]string, error) {
	if modified {
		return ModifiedFiles(ctx)
	}

	return StagedFiles(ctx)
}

// PackageDirsFromFiles derives unique Go package directory patterns from file paths.
// Non-Go files are ignored. Each directory is prefixed with "./" to form a valid
// package pattern accepted by golangci-lint and go test.
//
// Example: ["internal/pkg/checks/files.go", "cmd/root.go"]
//
//	→ ["./cmd", "./internal/pkg/checks"]
func PackageDirsFromFiles(files []string) []string {
	seen := map[string]struct{}{}
	dirs := []string{}

	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}

		dir := filepath.Join(".", filepath.Dir(f))
		if _, ok := seen[dir]; ok {
			continue
		}

		seen[dir] = struct{}{}
		dirs = append(dirs, dir)
	}

	slices.Sort(dirs)

	return dirs
}

// FilterByExt returns only the files whose extension matches one of exts.
func FilterByExt(files []string, exts ...string) []string {
	want := make(map[string]struct{}, len(exts))
	for _, e := range exts {
		want[e] = struct{}{}
	}

	result := []string{}

	for _, f := range files {
		if _, ok := want[filepath.Ext(f)]; ok {
			result = append(result, f)
		}
	}

	return result
}

// gitDiffFiles runs `git diff --name-only --diff-filter=ACM [extraFlags...]`
// and returns the resulting file paths.
func gitDiffFiles(ctx context.Context, extraFlags ...string) ([]string, error) {
	args := append([]string{"diff", "--name-only", "--diff-filter=ACM"}, extraFlags...)

	out, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("checks: git diff: %w", err)
	}

	return splitLines(string(out)), nil
}

func splitLines(s string) []string {
	lines := []string{}

	for line := range strings.SplitSeq(strings.TrimSpace(s), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}
