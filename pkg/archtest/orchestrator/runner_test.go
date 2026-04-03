package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunServiceCheck_Success(t *testing.T) {
	// Create temp directory for mock service
	tmpDir := t.TempDir()

	// Create a mise.toml with a simple passing arch:check task
	miseToml := `[tasks."arch:check"]
run = "exit 0"
`
	err := os.WriteFile(filepath.Join(tmpDir, "mise.toml"), []byte(miseToml), 0o644)
	if err != nil {
		t.Fatalf("Failed to create mise.toml: %v", err)
	}

	// This test validates that RunServiceCheck executes the command and returns success
	result := RunServiceCheck(tmpDir, "test-service")

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s", result.ExitCode, result.Output)
	}

	if result.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", result.ServiceName)
	}
}

func TestRunServiceCheck_Failure(t *testing.T) {
	// Create temp directory for mock service
	tmpDir := t.TempDir()

	// Create a mise.toml with a failing arch:check task
	miseToml := `[tasks."arch:check"]
run = "exit 42"
`
	err := os.WriteFile(filepath.Join(tmpDir, "mise.toml"), []byte(miseToml), 0o644)
	if err != nil {
		t.Fatalf("Failed to create mise.toml: %v", err)
	}

	// This test validates that RunServiceCheck captures non-zero exit codes
	// when mise run arch:check fails
	result := RunServiceCheck(tmpDir, "failing-service")

	if result.ServiceName != "failing-service" {
		t.Errorf("Expected service name 'failing-service', got '%s'", result.ServiceName)
	}

	if result.ExitCode != 42 {
		t.Errorf("Expected exit code 42, got %d. Output: %s", result.ExitCode, result.Output)
	}
}
