package config

import (
	"errors"
	"fmt"
	"slices"
)

var (
	// ErrSecretEmpty is returned when a required secret has no value.
	ErrSecretEmpty = errors.New("secret is empty")
	// ErrSecretPlaceholder is returned when a placeholder reaches a real environment.
	ErrSecretPlaceholder = errors.New("secret is a placeholder value")
)

// RequireSecret reports whether value is usable as the secret called name in env:
// empty is always an error; a placeholder is an error unless env is development or testing.
// Error messages carry name and env, never value.
func RequireSecret(env Environment, name, value string, placeholders ...string) error {
	if value == "" {
		return fmt.Errorf("%w: %s", ErrSecretEmpty, name)
	}

	if env != EnvironmentDevelopment && env != EnvironmentTesting && slices.Contains(placeholders, value) {
		return fmt.Errorf(
			"%w: %s in %s; set a real value in the encrypted overlay or the environment variable",
			ErrSecretPlaceholder, name, env,
		)
	}

	return nil
}
