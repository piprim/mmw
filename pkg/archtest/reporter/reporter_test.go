package reporter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestReporter_PrintHeader(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf)

	r.PrintHeader("Test Section")

	output := buf.String()
	if !strings.Contains(output, "Test Section") {
		t.Errorf("Expected header to contain 'Test Section', got: %s", output)
	}
	if !strings.Contains(output, "━") {
		t.Errorf("Expected header to contain separator line")
	}
}

func TestReporter_PrintCheckPass(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf)

	r.PrintCheck("test-check", "Test description", nil)

	output := buf.String()
	if !strings.Contains(output, "test-check") {
		t.Errorf("Expected check name in output")
	}
	if !strings.Contains(output, "✓ PASSED") {
		t.Errorf("Expected PASSED indicator")
	}
}

func TestReporter_PrintCheckFail(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf)
	err := fmt.Errorf("validation failed")

	r.PrintCheck("test-check", "Test description", err)

	output := buf.String()
	if !strings.Contains(output, "✗ FAILED") {
		t.Errorf("Expected FAILED indicator")
	}
	if !strings.Contains(output, "validation failed") {
		t.Errorf("Expected error message in output")
	}
}

func TestReporter_Summary(t *testing.T) {
	var buf bytes.Buffer
	r := NewReporter(&buf)

	// Track some checks
	r.PrintCheck("check1", "desc1", nil)
	r.PrintCheck("check2", "desc2", fmt.Errorf("error"))

	exitCode := r.Summary()

	if exitCode != 1 {
		t.Errorf("Expected exit code 1 when checks failed, got %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "Architecture validation failed") {
		t.Errorf("Expected failure message in summary")
	}
}
