package config

import (
	"context"
	"fmt"
	"os"
	"testing"
	"testing/fstest"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetViper cleans up viper's global state between tests
func resetViper() {
	viper.Reset()
}

// TestConfig is a test implementation of the Config interface
type TestConfig struct {
	Database    *TestDatabase `mapstructure:"database"`
	Environment string        `env:"APP_ENV, required"` // production, staging, testing, etc.
	AppName     string        `mapstructure:"app-name"` // Name of the application
}

type TestDatabase struct {
	User     string `mapstructure:"user"`
	Password string `env:"DB_PASSWORD, required"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Name     string `mapstructure:"name"`
}

// envString is a simple type that implements fmt.Stringer
type envString string

func (e envString) String() string {
	return string(e)
}

func (c *TestConfig) GetAppEnv() fmt.Stringer {
	return envString(c.Environment)
}

// setEnv sets env vars for the duration of a test, restoring originals on cleanup.
func setEnv(t *testing.T, envs map[string]string) {
	t.Helper()
	originals := make(map[string]string, len(envs))
	for k := range envs {
		originals[k] = os.Getenv(k)
	}
	t.Cleanup(func() {
		for k, orig := range originals {
			if orig != "" {
				os.Setenv(k, orig)
			} else {
				os.Unsetenv(k)
			}
		}
	})
	for k, v := range envs {
		require.NoError(t, os.Setenv(k, v))
	}
}

func TestContext_Fill(t *testing.T) {
	defaultFS := func(extra ...fstest.MapFS) fstest.MapFS {
		fs := fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte("[database]\nuser = \"rcv\"\nhost = \"localhost\"\nport = \"4332\"\nname = \"poc\"\n"),
			},
		}
		for _, extra := range extra {
			for k, v := range extra {
				fs[k] = v
			}
		}

		return fs
	}

	t.Run("returns error when required environment variables are missing", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		mockFS := defaultFS()

		t.Run("empty environment variables map", func(t *testing.T) {
			os.Unsetenv("APP_ENV")
			os.Unsetenv("DB_PASSWORD")
			err := NewContext(context.Background(), mockFS, "").Fill(&TestConfig{})
			assert.Error(t, err, "Fill() should return error when required environment variables are missing")
		})

		t.Run("missing APP_ENV", func(t *testing.T) {
			os.Unsetenv("APP_ENV")
			os.Unsetenv("DB_PASSWORD")
			err := NewContext(context.Background(), mockFS, "").Fill(&TestConfig{})
			assert.Error(t, err, "Fill() should return error when APP_ENV is missing")
		})

		t.Run("missing DB_PASSWORD", func(t *testing.T) {
			setEnv(t, map[string]string{"APP_ENV": "test"})
			os.Unsetenv("DB_PASSWORD")
			err := NewContext(context.Background(), mockFS, "").Fill(&TestConfig{})
			assert.Error(t, err, "Fill() should return error when DB_PASSWORD is missing")
		})
	})

	t.Run("succeeds with all variables and loads database config", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		mockFS := fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte("app-name = \"my-app\"\n\n[database]\nuser = \"rcv\"\nhost = \"localhost\"\nport = \"4332\"\nname = \"poc\"\n"),
			},
		}
		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "development"})

		config := &TestConfig{}
		err := NewContext(context.Background(), mockFS, "").Fill(config)

		require.NoError(t, err)
		require.NotNil(t, config.Database)
		assert.Equal(t, "rcv", config.Database.User)
		assert.Equal(t, "localhost", config.Database.Host)
		assert.Equal(t, "4332", config.Database.Port)
		assert.Equal(t, "poc", config.Database.Name)
		assert.Equal(t, "test_password", config.Database.Password)
		assert.Equal(t, "development", config.Environment)
		assert.Equal(t, "my-app", config.AppName)
	})

	t.Run("optional environment config missing does not cause error", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "nonexistent_env"})

		config := &TestConfig{}
		err := NewContext(context.Background(), defaultFS(), "").Fill(config)

		assert.NoError(t, err)
		require.NotNil(t, config.Database)
		assert.Equal(t, "rcv", config.Database.User)
	})

	t.Run("handles missing env vars without panicking", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		config := &TestConfig{}
		err := NewContext(context.Background(), defaultFS(), "").Fill(config)
		if err != nil {
			t.Logf("Fill() with no env vars failed as expected: %v", err)
		}
	})

	t.Run("uses real OS environment variables", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		setEnv(t, map[string]string{"DB_PASSWORD": "os_test_password", "APP_ENV": "default"})

		config := &TestConfig{}
		err := NewContext(context.Background(), defaultFS(), "").Fill(config)

		require.NoError(t, err)
		require.NotNil(t, config.Database)
		assert.Equal(t, "os_test_password", config.Database.Password)
		assert.Equal(t, "default", config.Environment)
		assert.Equal(t, "rcv", config.Database.User)
		assert.Equal(t, "localhost", config.Database.Host)
		assert.Equal(t, "4332", config.Database.Port)
		assert.Equal(t, "poc", config.Database.Name)
	})

	t.Run("merges environment config over default", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		mockFS := fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte("app-name = \"default-app\"\n\n[database]\nuser = \"rcv\"\nhost = \"localhost\"\nport = \"4332\"\nname = \"poc\"\n"),
			},
			"configs/testing.toml": &fstest.MapFile{
				Data: []byte("app-name = \"testing-app\"\n\n[database]\nuser = \"test_user\"\nhost = \"test.example.com\"\nport = \"5433\"\nname = \"testdb\"\n"),
			},
		}
		setEnv(t, map[string]string{"DB_PASSWORD": "merged_password", "APP_ENV": "testing"})

		config := new(TestConfig)
		err := NewContext(context.Background(), mockFS, "").Fill(config)

		require.NoError(t, err)
		require.NotNil(t, config.Database)
		assert.Equal(t, "test_user", config.Database.User)
		assert.Equal(t, "test.example.com", config.Database.Host)
		assert.Equal(t, "5433", config.Database.Port)
		assert.Equal(t, "testdb", config.Database.Name)
		assert.Equal(t, "merged_password", config.Database.Password)
		assert.Equal(t, "testing", config.Environment)
		assert.Equal(t, "testing-app", config.AppName)
	})

	t.Run("partial environment config only overrides specified fields", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		mockFS := defaultFS(fstest.MapFS{
			"configs/staging.toml": &fstest.MapFile{
				Data: []byte("[database]\nhost = \"partial.example.com\"\n"),
			},
		})
		setEnv(t, map[string]string{"DB_PASSWORD": "partial_password", "APP_ENV": "staging"})

		config := &TestConfig{}
		err := NewContext(context.Background(), mockFS, "").Fill(config)

		require.NoError(t, err)
		require.NotNil(t, config.Database)
		assert.Equal(t, "partial.example.com", config.Database.Host)
		assert.Equal(t, "rcv", config.Database.User)
		assert.Equal(t, "4332", config.Database.Port)
		assert.Equal(t, "poc", config.Database.Name)
		assert.Equal(t, "partial_password", config.Database.Password)
		assert.Equal(t, "staging", config.Environment)
	})

	t.Run("returns error when required OS environment variables are missing", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		origDBPassword := os.Getenv("DB_PASSWORD")
		origAppEnv := os.Getenv("APP_ENV")
		t.Cleanup(func() {
			if origDBPassword != "" {
				os.Setenv("DB_PASSWORD", origDBPassword)
			} else {
				os.Unsetenv("DB_PASSWORD")
			}
			if origAppEnv != "" {
				os.Setenv("APP_ENV", origAppEnv)
			} else {
				os.Unsetenv("APP_ENV")
			}
		})
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("APP_ENV")

		err := NewContext(context.Background(), defaultFS(), "").Fill(&TestConfig{})
		assert.Error(t, err)
	})

	t.Run("returns error when default config file is missing", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "default"})

		err := NewContext(context.Background(), fstest.MapFS{}, "").Fill(&TestConfig{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read the default configuration")
	})

	t.Run("AppName loaded from default config", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		mockFS := fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte("app-name = \"service-name\"\n\n[database]\nuser = \"rcv\"\nhost = \"localhost\"\nport = \"4332\"\nname = \"poc\"\n"),
			},
		}
		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "production"})

		config := &TestConfig{}
		require.NoError(t, NewContext(context.Background(), mockFS, "").Fill(config))
		assert.Equal(t, "service-name", config.AppName)
		assert.Equal(t, "production", config.Environment)
	})

	t.Run("AppName empty when not specified in TOML", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "development"})

		config := &TestConfig{}
		require.NoError(t, NewContext(context.Background(), defaultFS(), "").Fill(config))
		assert.Empty(t, config.AppName)
		assert.Equal(t, "development", config.Environment)
	})

	t.Run("AppName overridden by environment-specific config", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		mockFS := defaultFS(fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte("app-name = \"default-service\"\n\n[database]\nuser = \"rcv\"\nhost = \"localhost\"\nport = \"4332\"\nname = \"poc\"\n"),
			},
			"configs/production.toml": &fstest.MapFile{
				Data: []byte("app-name = \"production-service\""),
			},
		})
		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "production"})

		config := &TestConfig{}
		require.NoError(t, NewContext(context.Background(), mockFS, "").Fill(config))
		assert.Equal(t, "production-service", config.AppName)
		assert.Equal(t, "production", config.Environment)
	})

	t.Run("AppName handles special characters", func(t *testing.T) {
		resetViper()
		t.Cleanup(resetViper)

		mockFS := fstest.MapFS{
			"configs/default.toml": &fstest.MapFile{
				Data: []byte("app-name = \"my-app_v1.0\"\n\n[database]\nuser = \"rcv\"\nhost = \"localhost\"\nport = \"4332\"\nname = \"poc\"\n"),
			},
		}
		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "staging"})

		config := &TestConfig{}
		require.NoError(t, NewContext(context.Background(), mockFS, "").Fill(config))
		assert.Equal(t, "my-app_v1.0", config.AppName)
		assert.Equal(t, "staging", config.Environment)
	})
}

func TestEnvUnmarshal(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	t.Run("with envs map", func(t *testing.T) {
		setEnv(t, map[string]string{
			"DB_PASSWORD": "secret_pass",
			"APP_ENV":     "production",
		})

		config := &TestConfig{}
		err := envUnmarshal(ctx, config, nil, "")
		require.NoError(t, err)

		assert.Equal(t, "production", config.Environment)
		assert.Equal(t, "secret_pass", config.Database.Password)
	})

	t.Run("with nil envs uses OS environment", func(t *testing.T) {
		config := &TestConfig{}
		err := envUnmarshal(ctx, config, nil, "")
		if err != nil {
			t.Logf("envUnmarshal() with nil envs failed as expected: %v", err)
		}
	})
}
