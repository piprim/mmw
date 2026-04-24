package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverServices(t *testing.T) {
	t.Run("returns all dirs and flags those with the arch:check task", func(t *testing.T) {
		tmpDir := t.TempDir()
		servicesDir := filepath.Join(tmpDir, "services")

		service1 := filepath.Join(servicesDir, "service1")
		service2 := filepath.Join(servicesDir, "service2")
		notService := filepath.Join(servicesDir, "README.md")

		os.MkdirAll(service1, 0o755)
		os.MkdirAll(service2, 0o755)
		os.WriteFile(notService, []byte("test"), 0o644)

		// service1 has the arch:check task; service2 does not.
		os.WriteFile(filepath.Join(service1, "mise.toml"), []byte(`
[tasks."arch:check"]
run = "arch-go check"
`), 0o644)

		services, err := DiscoverServices(servicesDir, "arch:check")
		if err != nil {
			t.Fatalf("DiscoverServices failed: %v", err)
		}

		if len(services) != 2 {
			t.Errorf("expected 2 services (all dirs), got %d", len(services))
		}

		found := false
		for _, svc := range services {
			if svc.Name == "service1" {
				found = true
				if !svc.HasArchCheck {
					t.Errorf("expected service1 to have arch:check task")
				}
			}
			if svc.Name == "service2" && svc.HasArchCheck {
				t.Errorf("expected service2 to NOT have arch:check task")
			}
		}
		if !found {
			t.Errorf("service1 not found in results")
		}
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		nonExistent := filepath.Join(t.TempDir(), "nonexistent")

		services, err := DiscoverServices(nonExistent, "arch:check")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
		if services != nil {
			t.Error("expected nil services on error")
		}
	})
}
