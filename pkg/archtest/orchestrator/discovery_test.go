package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverServices(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	servicesDir := filepath.Join(tmpDir, "services")

	// Create mock services
	service1 := filepath.Join(servicesDir, "service1")
	service2 := filepath.Join(servicesDir, "service2")
	notService := filepath.Join(servicesDir, "README.md")

	os.MkdirAll(service1, 0o755)
	os.MkdirAll(service2, 0o755)
	os.WriteFile(notService, []byte("test"), 0o644)

	// Create mise.toml for service1
	os.WriteFile(filepath.Join(service1, "mise.toml"), []byte(`
[tasks."arch:check"]
run = "arch-go check"
`), 0o644)

	services, err := DiscoverServices(servicesDir, "arch:check")
	if err != nil {
		t.Fatalf("DiscoverServices failed: %v", err)
	}

	// DiscoverServices returns all directories; HasArchCheck flags those with the task.
	if len(services) != 2 {
		t.Errorf("Expected 2 services (all dirs), got %d", len(services))
	}

	// Find service1 and assert it has HasArchCheck=true
	found := false
	for _, svc := range services {
		if svc.Name == "service1" {
			found = true
			if !svc.HasArchCheck {
				t.Errorf("Expected service1 to have arch:check task")
			}
		}
		if svc.Name == "service2" && svc.HasArchCheck {
			t.Errorf("Expected service2 to NOT have arch:check task")
		}
	}
	if !found {
		t.Errorf("service1 not found in results")
	}
}

func TestDiscoverServices_NoServicesDir(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "nonexistent")

	services, err := DiscoverServices(nonExistent, "arch:check")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
	if services != nil {
		t.Error("Expected nil services on error")
	}
}
