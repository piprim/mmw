package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentValues(t *testing.T) {
	values := EnvironmentValues()
	assert.Len(t, values, 4)
	assert.Contains(t, values, EnvironmentDevelopment)
	assert.Contains(t, values, EnvironmentStaging)
	assert.Contains(t, values, EnvironmentProduction)
	assert.Contains(t, values, EnvironmentTesting)
}

func TestEnvironment_String(t *testing.T) {
	tests := []struct {
		env      Environment
		expected string
	}{
		{EnvironmentDevelopment, "development"},
		{EnvironmentStaging, "staging"},
		{EnvironmentProduction, "production"},
		{EnvironmentTesting, "testing"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.env.String())
		})
	}
}

func TestEnvironment_IsValid(t *testing.T) {
	tests := []struct {
		env   Environment
		valid bool
	}{
		{EnvironmentDevelopment, true},
		{EnvironmentStaging, true},
		{EnvironmentProduction, true},
		{EnvironmentTesting, true},
		{Environment("unknown"), false},
		{Environment(""), false},
		{Environment("Development"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.env), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.env.IsValid())
		})
	}
}

func TestParseEnvironment(t *testing.T) {
	tests := []struct {
		input   string
		want    Environment
		wantErr bool
	}{
		{"development", EnvironmentDevelopment, false},
		{"staging", EnvironmentStaging, false},
		{"production", EnvironmentProduction, false},
		{"testing", EnvironmentTesting, false},
		{"unknown", Environment(""), true},
		{"", Environment(""), true},
		{"Development", Environment(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseEnvironment(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidEnvironment))
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEnvironment_MarshalText(t *testing.T) {
	env := EnvironmentProduction
	b, err := env.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("production"), b)
}

func TestEnvironment_UnmarshalText(t *testing.T) {
	t.Run("valid value", func(t *testing.T) {
		var env Environment
		err := env.UnmarshalText([]byte("staging"))
		require.NoError(t, err)
		assert.Equal(t, EnvironmentStaging, env)
	})

	t.Run("invalid value", func(t *testing.T) {
		var env Environment
		err := env.UnmarshalText([]byte("invalid"))
		assert.Error(t, err)
	})
}

func TestEnvironment_AppendText(t *testing.T) {
	env := EnvironmentTesting
	b, err := env.AppendText([]byte("prefix-"))
	require.NoError(t, err)
	assert.Equal(t, []byte("prefix-testing"), b)
}

func TestEnvironment_AppendText_EmptyPrefix(t *testing.T) {
	env := EnvironmentDevelopment
	b, err := env.AppendText(nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("development"), b)
}

func TestEnvironment_IsDev(t *testing.T) {
	tests := []struct {
		env   Environment
		isDev bool
	}{
		{EnvironmentDevelopment, true},
		{EnvironmentStaging, false},
		{EnvironmentProduction, false},
		{EnvironmentTesting, false},
		{Environment("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.env), func(t *testing.T) {
			assert.Equal(t, tt.isDev, tt.env.IsDev())
		})
	}
}
