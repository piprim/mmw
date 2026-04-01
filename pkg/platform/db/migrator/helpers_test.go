package migrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnaccentString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"cafe", "cafe"},
		{"café", "cafe"},
		{"naïve", "naive"},
		{"résumé", "resume"},
		{"Ångström", "Angstrom"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := unaccentString(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple words",
			input:    "create users table",
			expected: "create-users-table",
		},
		{
			name:     "already dashed",
			input:    "add-index",
			expected: "add-index",
		},
		{
			name:     "accented chars",
			input:    "ajout résumé",
			expected: "ajout-resume",
		},
		{
			name:     "special chars replaced",
			input:    "add column@users",
			expected: "add-column-users",
		},
		{
			name:     "consecutive separators collapsed",
			input:    "drop   table",
			expected: "drop-table",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:     "with extension preserved",
			input:    "create.sql",
			expected: "create.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeFileName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
