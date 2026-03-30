package platform_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/piprim/mmw/platform"
)

func TestDomainError_Error_ReturnsMessage(t *testing.T) {
	err := &platform.DomainError{Code: 42, Message: "something went wrong"}

	if err.Error() != "something went wrong" {
		t.Errorf("Error() = %q, want %q", err.Error(), "something went wrong")
	}
}

func TestDomainError_AsType_ThroughWrapping(t *testing.T) {
	original := &platform.DomainError{Code: 1, Message: "original"}
	wrapped := fmt.Errorf("context: %w", original)

	got, ok := errors.AsType[*platform.DomainError](wrapped)

	if !ok {
		t.Fatal("errors.AsType failed to find *platform.DomainError through wrapping")
	}

	if got.Code != 1 {
		t.Errorf("Code = %v, want 1", got.Code)
	}
}
