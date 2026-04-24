package checks

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
)

const maxFileSize = 512_000 // 500 KB

const maxLineBufBytes = 1024 * 1024 // 1 MB — maximum line length for bufio.Scanner

type filesChecker struct{}

// NewFilesChecker returns a Checker (which also implements Fixer) that validates
// trailing whitespace, missing EOF newline, and file size.
func NewFilesChecker() Checker {
	return &filesChecker{}
}

func (*filesChecker) Name() string {
	return "files"
}

// Check validates each target file. When targets is empty it falls back to
// all git-tracked files under the working directory.
func (c *filesChecker) Check(ctx context.Context, targets []string) (Result, error) {
	files, err := resolveTargets(ctx, targets)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		CheckerName: c.Name(),
		Violations:  []Violation{},
	}

	for _, path := range files {
		vs, err := checkFileContent(path)
		if err != nil {
			return Result{}, fmt.Errorf("checks: files: %w", err)
		}

		result.Violations = append(result.Violations, vs...)
	}

	return result, nil
}

// Fix rewrites each target file in-place, stripping trailing whitespace and
// appending a missing EOF newline. Size violations are never auto-fixed.
func (*filesChecker) Fix(ctx context.Context, targets []string) error {
	files, err := resolveTargets(ctx, targets)
	if err != nil {
		return err
	}

	for _, path := range files {
		if err := fixFileContent(path); err != nil {
			return fmt.Errorf("checks: files fix %s: %w", path, err)
		}
	}

	return nil
}

func checkFileContent(path string) ([]Violation, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if info.IsDir() {
		return nil, nil // submodules and directories appear in git diff output; skip them
	}

	vs := []Violation{}

	if info.Size() > maxFileSize {
		return []Violation{{
			File:    path,
			Message: fmt.Sprintf("file size %d bytes exceeds 500 KB limit", info.Size()),
		}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	lineNum := 0
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, maxLineBufBytes), maxLineBufBytes)

	for scanner.Scan() {
		lineNum++

		line := scanner.Text()
		if strings.TrimRight(line, " \t") != line {
			vs = append(vs, Violation{
				File:    path,
				Line:    lineNum,
				Message: "trailing whitespace",
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	if len(data) > 0 && data[len(data)-1] != '\n' {
		vs = append(vs, Violation{
			File:    path,
			Message: "missing newline at end of file",
		})
	}

	return vs, nil
}

func fixFileContent(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if info.IsDir() {
		return nil
	}

	if info.Size() > maxFileSize {
		// Size violations cannot be auto-fixed; skip silently.
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	var buf bytes.Buffer

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, maxLineBufBytes), maxLineBufBytes)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(&buf, strings.TrimRight(line, " \t"))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
