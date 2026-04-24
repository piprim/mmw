package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunServiceCheck(t *testing.T) {
	t.Run("returns exit code 0 and service name on success", func(t *testing.T) {
		tmpDir := t.TempDir()

		miseToml := `[tasks."arch:check"]
run = "exit 0"
`
		if err := os.WriteFile(filepath.Join(tmpDir, "mise.toml"), []byte(miseToml), 0o644); err != nil {
			t.Fatalf("failed to create mise.toml: %v", err)
		}

		result := RunServiceCheck(tmpDir, "test-service")

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d. Output: %s", result.ExitCode, result.Output)
		}
		if result.ServiceName != "test-service" {
			t.Errorf("expected service name 'test-service', got '%s'", result.ServiceName)
		}
	})

	t.Run("captures non-zero exit code on failure", func(t *testing.T) {
		tmpDir := t.TempDir()

		miseToml := `[tasks."arch:check"]
run = "exit 42"
`
		if err := os.WriteFile(filepath.Join(tmpDir, "mise.toml"), []byte(miseToml), 0o644); err != nil {
			t.Fatalf("failed to create mise.toml: %v", err)
		}

		result := RunServiceCheck(tmpDir, "failing-service")

		if result.ServiceName != "failing-service" {
			t.Errorf("expected service name 'failing-service', got '%s'", result.ServiceName)
		}
		if result.ExitCode != 42 {
			t.Errorf("expected exit code 42, got %d. Output: %s", result.ExitCode, result.Output)
		}
	})
}
