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
	if strings.Contains(string(content)+"\n", newEntry+"\n") {
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
	if err := os.WriteFile(goWorkPath, []byte(updated), 0o600); err != nil {
		return fmt.Errorf("write go.work: %w", err)
	}

	return nil
}

// UpdateMiseToml adds the module's tasks to the root mise.toml: test and
// contract always, integration only with a database, run only with gRPC.
func UpdateMiseToml(repoRoot, name string, database, grpc bool) error {
	misePath := filepath.Join(repoRoot, "mise.toml")
	content, err := os.ReadFile(misePath)
	if err != nil {
		return fmt.Errorf("read mise.toml: %w", err)
	}

	marker := fmt.Sprintf(`[tasks."%s:test"]`, name)
	if strings.Contains(string(content), marker) {
		return nil // idempotent
	}

	task := func(suffix, desc, run string) string {
		return fmt.Sprintf("\n[tasks.\"%s%s\"]\ndescription = \"%s %s module\"\nrun = \"cd modules/%s && mise run %s\"\n",
			name, suffix, desc, name, name, run)
	}
	addition := task(":test", "Run unit tests for", "test:unit") +
		task(":test:contract", "Run contract tests for", "test:contract")
	if database {
		addition += task(":test:integration", "Run integration tests for", "test:integration")
	}
	if grpc {
		addition += task(":run", "Run the", "run")
	}

	//nolint:gosec // G703: Path traversal is safe here
	if err := os.WriteFile(misePath, append(content, []byte(addition)...), 0o600); err != nil {
		return fmt.Errorf("write mise.toml: %w", err)
	}

	return nil
}
