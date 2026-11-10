package config

import (
	"errors"
	"strings"
	"testing"
)

func TestRequireSecret(t *testing.T) {
	placeholders := []string{"rcv", "fake-secret"}

	tests := []struct {
		name    string
		env     Environment
		value   string
		wantErr error
	}{
		{"empty in development", EnvironmentDevelopment, "", ErrSecretEmpty},
		{"placeholder in development", EnvironmentDevelopment, "rcv", nil},
		{"placeholder in testing", EnvironmentTesting, "fake-secret", nil},
		{"placeholder in staging", EnvironmentStaging, "rcv", ErrSecretPlaceholder},
		{"placeholder in production", EnvironmentProduction, "fake-secret", ErrSecretPlaceholder},
		{"real value in production", EnvironmentProduction, "s3cr3t", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RequireSecret(tt.env, "db password", tt.value, placeholders...); !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("error message does not contain the value", func(t *testing.T) {
		err := RequireSecret(EnvironmentProduction, "db password", "fake-secret", placeholders...)
		if err == nil || strings.Contains(err.Error(), "fake-secret") {
			t.Errorf("err = %v, must exist and not carry the value", err)
		}
	})

	t.Run("error message names the secret and the environment", func(t *testing.T) {
		err := RequireSecret(EnvironmentProduction, "db password", "fake-secret", placeholders...)
		if err == nil || !strings.Contains(err.Error(), "db password in production") {
			t.Errorf("err = %v, want it to name the secret and the environment", err)
		}
	})
}
