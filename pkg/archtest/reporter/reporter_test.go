package reporter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestReporter(t *testing.T) {
	t.Run("PrintHeader contains title and separator", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf)

		r.PrintHeader("Test Section")

		output := buf.String()
		if !strings.Contains(output, "Test Section") {
			t.Errorf("expected header to contain 'Test Section', got: %s", output)
		}
		if !strings.Contains(output, "━") {
			t.Errorf("expected header to contain separator line")
		}
	})

	t.Run("PrintCheck shows passed when no error", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf)

		r.PrintCheck("test-check", "Test description", nil)

		output := buf.String()
		if !strings.Contains(output, "test-check") {
			t.Errorf("expected check name in output")
		}
		if !strings.Contains(output, "✓ PASSED") {
			t.Errorf("expected PASSED indicator")
		}
	})

	t.Run("PrintCheck shows failed and error message", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf)
		err := fmt.Errorf("validation failed")

		r.PrintCheck("test-check", "Test description", err)

		output := buf.String()
		if !strings.Contains(output, "✗ FAILED") {
			t.Errorf("expected FAILED indicator")
		}
		if !strings.Contains(output, "validation failed") {
			t.Errorf("expected error message in output")
		}
	})

	t.Run("Summary returns exit code 1 when checks failed", func(t *testing.T) {
		var buf bytes.Buffer
		r := NewReporter(&buf)

		r.PrintCheck("check1", "desc1", nil)
		r.PrintCheck("check2", "desc2", fmt.Errorf("error"))

		exitCode := r.Summary()

		if exitCode != 1 {
			t.Errorf("expected exit code 1 when checks failed, got %d", exitCode)
		}

		output := buf.String()
		if !strings.Contains(output, "Architecture validation failed") {
			t.Errorf("expected failure message in summary")
		}
	})
}
