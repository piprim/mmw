package oglos

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvMap(t *testing.T) {
	t.Run("returns map with environment variables", func(t *testing.T) {
		envMap := EnvMap()

		if envMap == nil {
			t.Fatal("EnvMap() returned nil")
		}

		// Should have at least some environment variables
		if len(envMap) == 0 {
			t.Error("EnvMap() returned empty map, expected environment variables")
		}
	})

	t.Run("contains expected environment variables", func(t *testing.T) {
		// Set a test environment variable
		testKey := "OGLOS_TEST_VAR"
		testValue := "test_value_123"
		os.Setenv(testKey, testValue)
		defer os.Unsetenv(testKey)

		envMap := EnvMap()

		if val, exists := envMap[testKey]; !exists {
			t.Errorf("EnvMap() missing expected key %q", testKey)
		} else if val != testValue {
			t.Errorf("EnvMap()[%q] = %q, want %q", testKey, val, testValue)
		}
	})

	t.Run("handles environment variables with equals sign in value", func(t *testing.T) {
		testKey := "OGLOS_EQUALS_TEST"
		testValue := "value=with=equals"
		os.Setenv(testKey, testValue)
		defer os.Unsetenv(testKey)

		envMap := EnvMap()

		if val, exists := envMap[testKey]; !exists {
			t.Errorf("EnvMap() missing key %q", testKey)
		} else if val != testValue {
			t.Errorf("EnvMap()[%q] = %q, want %q (should preserve equals signs in value)", testKey, val, testValue)
		}
	})

	t.Run("handles empty environment variable value", func(t *testing.T) {
		testKey := "OGLOS_EMPTY_VAR"
		os.Setenv(testKey, "")
		defer os.Unsetenv(testKey)

		envMap := EnvMap()

		if val, exists := envMap[testKey]; !exists {
			t.Errorf("EnvMap() missing key %q", testKey)
		} else if val != "" {
			t.Errorf("EnvMap()[%q] = %q, want empty string", testKey, val)
		}
	})

	t.Run("map size matches os.Environ length", func(t *testing.T) {
		envMap := EnvMap()
		environSlice := os.Environ()

		if len(envMap) != len(environSlice) {
			t.Errorf("EnvMap() length = %d, os.Environ() length = %d, should match", len(envMap), len(environSlice))
		}
	})

	t.Run("all os.Environ entries are present in map", func(t *testing.T) {
		envMap := EnvMap()
		environSlice := os.Environ()

		for _, entry := range environSlice {
			// Parse the entry manually to compare
			var key string
			for i := 0; i < len(entry); i++ {
				if entry[i] == '=' {
					key = entry[:i]
					break
				}
			}

			if key == "" {
				continue // Skip malformed entries
			}

			if _, exists := envMap[key]; !exists {
				t.Errorf("EnvMap() missing key %q from os.Environ()", key)
			}
		}
	})

	t.Run("all map entries are the same as os.getenv", func(t *testing.T) {
		envMap := EnvMap()

		for key, value := range envMap {
			assert.Equal(t, os.Getenv(key), value, "os.Getenv does not have the value", value, "for key", key)
		}
	})
}
