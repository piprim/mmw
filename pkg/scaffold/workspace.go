package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UpdateGoWork adds the new module path to go.work if not already present.
func UpdateGoWork(repoRoot, name string) error {
	goWorkPath := filepath.Join(repoRoot, "go.work")
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		return fmt.Errorf("read go.work: %w", err)
	}

	newEntry := "\t./modules/" + name
	if strings.Contains(string(content), newEntry) {
		return nil // already present, idempotent
	}

	// Locate the use block, then find its closing ")" to insert before it.
	// Searching from the "use (" position avoids accidentally matching a
	// closing ")" in an earlier replace () block.
	useIdx := bytes.Index(content, []byte("use ("))
	if useIdx == -1 {
		return errors.New("could not find use block in go.work")
	}

	closeIdx := strings.Index(string(content)[useIdx:], "\n)")
	if closeIdx == -1 {
		return errors.New("malformed use block in go.work: missing closing )")
	}

	pos := useIdx + closeIdx
	updated := string(content)[:pos] + "\n" + newEntry + string(content)[pos:]

	//nolint:gosec // G703: Path traversal is safe here
	if err := os.WriteFile(goWorkPath, []byte(updated), 0600); err != nil {
		return fmt.Errorf("write go.work: %w", err)
	}

	return nil
}

// UpdateMiseToml adds test/build tasks for the new module to poc/mise.toml.
func UpdateMiseToml(repoRoot, name string) error {
	misePath := filepath.Join(repoRoot, "mise.toml")
	content, err := os.ReadFile(misePath)
	if err != nil {
		return fmt.Errorf("read mise.toml: %w", err)
	}

	marker := fmt.Sprintf(`[tasks."%s:test"]`, name)
	if strings.Contains(string(content), marker) {
		return nil // idempotent
	}

	addition := fmt.Sprintf(`
[tasks."%s:test"]
description = "Run unit tests for %s module"
run = "cd modules/%s && mise run test"

[tasks."%s:test:integration"]
description = "Run integration tests for %s module"
run = "cd modules/%s && mise run test:integration"

[tasks."%s:test:contract"]
description = "Run contract tests for %s module"
run = "cd modules/%s && mise run test:contract"
`, name, name, name, name, name, name, name, name, name)

	//nolint:gosec // G703: Path traversal is safe here
	if err := os.WriteFile(misePath, append(content, []byte(addition)...), 0600); err != nil {
		return fmt.Errorf("write mise.toml: %w", err)
	}

	return nil
}
