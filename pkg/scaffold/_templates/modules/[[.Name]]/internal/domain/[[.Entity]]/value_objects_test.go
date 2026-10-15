package [[.Entity]]_test

import (
	"errors"
	"strings"
	"testing"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

func TestNewLabel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "one character", input: "a"},
		{name: "exactly 200 runes", input: strings.Repeat("é", [[.Entity]].LabelMaxLen)},
		{name: "empty", input: "", wantErr: [[.Entity]].ErrInvalidLabel},
		{name: "201 runes", input: strings.Repeat("é", [[.Entity]].LabelMaxLen+1), wantErr: [[.Entity]].ErrInvalidLabel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := [[.Entity]].NewLabel(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewLabel() error = %v, want %v", err, tt.wantErr)
			}
			if err == nil && got.String() != tt.input {
				t.Errorf("String() = %q, want %q", got.String(), tt.input)
			}
		})
	}
}

func TestReconstituteLabel(t *testing.T) {
	t.Run("trusts a value the rule would reject", func(t *testing.T) {
		if got := [[.Entity]].ReconstituteLabel("").String(); got != "" {
			t.Errorf("String() = %q, want empty", got)
		}
	})
}
