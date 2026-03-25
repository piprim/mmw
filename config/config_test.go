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
	Database    *Database `mapstructure:"database"`
	Environment string    `env:"APP_ENV, required"` // production, staging, testing, etc.
	AppName     string    `mapstructure:"app-name"` // Name of the application
}

type Database struct {
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

func TestContext_Fill_MissingEnvironmentVariables(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	t.Run("empty environment variables map", func(t *testing.T) {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("DB_PASSWORD")
		configCtx := NewContext(ctx, mockFS, "")
		config := &TestConfig{}
		err := configCtx.Fill(config)
		assert.Error(t, err, "Fill() should return error when required environment variables are missing")
	})

	t.Run("missing APP_ENV", func(t *testing.T) {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("DB_PASSWORD")
		configCtx := NewContext(ctx, mockFS, "")
		config := &TestConfig{}
		err := configCtx.Fill(config)
		assert.Error(t, err, "Fill() should return error when APP_ENV is missing")
	})

	t.Run("missing DB_PASSWORD", func(t *testing.T) {
		setEnv(t, map[string]string{"APP_ENV": "test"})
		os.Unsetenv("DB_PASSWORD")
		configCtx := NewContext(ctx, mockFS, "")
		config := &TestConfig{}
		err := configCtx.Fill(config)
		assert.Error(t, err, "Fill() should return error when DB_PASSWORD is missing")
	})
}

func TestContext_Fill_Success(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`app-name = "my-app"

[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "test_password",
		"APP_ENV":     "development",
	})

	config := &TestConfig{}
	err := NewContext(ctx, mockFS, "").Fill(config)

	require.NoError(t, err, "Fill() should not return error with valid environment variables")
	require.NotNil(t, config.Database, "Config should have non-nil Database")

	// Verify database config was loaded from default config
	assert.Equal(t, "rcv", config.Database.User, "Database.User should match default config")
	assert.Equal(t, "localhost", config.Database.Host, "Database.Host should match default config")
	assert.Equal(t, "4332", config.Database.Port, "Database.Port should match default config")
	assert.Equal(t, "poc", config.Database.Name, "Database.Name should match default config")

	// Verify environment variable was processed
	assert.Equal(t, "test_password", config.Database.Password, "Database.Password should match environment variable")
	assert.Equal(t, "development", config.Environment, "Config.Environment should match environment variable")

	// Verify AppName was loaded from TOML (root level)
	assert.Equal(t, "my-app", config.AppName, "AppName should match default config")
}

func TestContext_Fill_OptionalEnvironmentConfig(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	// Environment-specific config is optional and should not cause an error
	setEnv(t, map[string]string{
		"DB_PASSWORD": "test_password",
		"APP_ENV":     "nonexistent_env",
	})

	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	assert.NoError(t, err, "Fill() should not error when environment-specific config doesn't exist")
	require.NotNil(t, config.Database, "Config should have non-nil Database")

	// Verify default config was loaded
	assert.Equal(t, "rcv", config.Database.User, "Database.User should match default config when env-specific config doesn't exist")
}

func TestContext_Fill_WithNilEnvs(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	// When env vars are not set, Fill should use actual OS environment variables
	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)
	// We can't reliably test this without manipulating OS environment
	// Just verify the function handles missing env vars without panicking
	if err != nil {
		// Expected to fail if OS env vars are not set
		t.Logf("Fill() with no env vars failed as expected when env vars not set: %v", err)
	}
}

func TestContext_Fill_WithRealOSEnvironment(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "os_test_password",
		"APP_ENV":     "default",
	})

	// Fill config using OS environment
	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	require.NoError(t, err, "Fill() should not return error with OS environment variables set")
	require.NotNil(t, config.Database, "Config should have non-nil Database")

	// Verify OS environment variables were used
	assert.Equal(t, "os_test_password", config.Database.Password, "Database.Password should match OS environment variable")
	assert.Equal(t, "default", config.Environment, "Config.Environment should match OS environment variable")

	// Verify default config was still loaded
	assert.Equal(t, "rcv", config.Database.User, "Database.User should match default config")
	assert.Equal(t, "localhost", config.Database.Host, "Database.Host should match default config")
	assert.Equal(t, "4332", config.Database.Port, "Database.Port should match default config")
	assert.Equal(t, "poc", config.Database.Name, "Database.Name should match default config")
}

func TestContext_Fill_ConfigMergingWithEnvironmentFile(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	// Mock the filesystem with default and testing configs
	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`app-name = "default-app"

[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
		"configs/testing.toml": &fstest.MapFile{
			Data: []byte(`app-name = "testing-app"

[database]
user = "test_user"
host = "test.example.com"
port = "5433"
name = "testdb"
`),
		},
	}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "merged_password",
		"APP_ENV":     "testing",
	})

	configCtx := NewContext(ctx, mockFS, "")
	config := new(TestConfig)
	err := configCtx.Fill(config)

	require.NoError(t, err, "Fill() should not return error with environment-specific config file")
	require.NotNil(t, config.Database, "Config should have non-nil Database")

	// Verify that environment-specific config overrides default config
	assert.Equal(t, "test_user", config.Database.User, "Database.User should be from testing config, not default")
	assert.Equal(t, "test.example.com", config.Database.Host, "Database.Host should be from testing config, not default")
	assert.Equal(t, "5433", config.Database.Port, "Database.Port should be from testing config, not default")
	assert.Equal(t, "testdb", config.Database.Name, "Database.Name should be from testing config, not default")

	// Verify environment variable was still processed
	assert.Equal(t, "merged_password", config.Database.Password, "Database.Password should match environment variable")
	assert.Equal(t, "testing", config.Environment, "Config.Environment should match environment variable")

	// Verify AppName was overridden by testing config
	assert.Equal(t, "testing-app", config.AppName, "AppName should be from testing config, not default")
}

func TestContext_Fill_ConfigMergingPartialOverride(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	// Mock the filesystem with default and partial configs
	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`app-name = "default-app"

[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
		"configs/staging.toml": &fstest.MapFile{
			Data: []byte(`[database]
host = "partial.example.com"
`),
		},
	}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "partial_password",
		"APP_ENV":     "staging",
	})

	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	require.NoError(t, err, "Fill() should not return error with partial environment-specific config")
	require.NotNil(t, config.Database, "Config should have non-nil Database")

	// Verify that only specified fields are overridden
	assert.Equal(t, "partial.example.com", config.Database.Host, "Database.Host should be overridden from partial config")

	// Verify that non-specified fields still come from default config
	assert.Equal(t, "rcv", config.Database.User, "Database.User should still be from default config")
	assert.Equal(t, "4332", config.Database.Port, "Database.Port should still be from default config")
	assert.Equal(t, "poc", config.Database.Name, "Database.Name should still be from default config")

	// Verify environment variable was processed
	assert.Equal(t, "partial_password", config.Database.Password, "Database.Password should match environment variable")
	assert.Equal(t, "staging", config.Environment, "Config.Environment should match environment variable")

	// Verify AppName was NOT overridden (remains from default)
	assert.Equal(t, "default-app", config.AppName, "AppName should still be from default config when not specified in partial")
}

func TestContext_Fill_WithRealOSEnvironment_MissingRequired(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	// Save and unset required environment variables
	origDBPassword := os.Getenv("DB_PASSWORD")
	origAppEnv := os.Getenv("APP_ENV")
	defer func() {
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
	}()

	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("APP_ENV")

	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	assert.Error(t, err, "Fill() should return error when required OS environment variables are missing")
}

func TestContext_Fill_MissingDefaultConfig(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	// Empty filesystem - no default.toml
	mockFS := fstest.MapFS{}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "test_password",
		"APP_ENV":     "default",
	})

	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	assert.Error(t, err, "Fill() should return error when default config is missing")
	assert.Contains(t, err.Error(), "failed to read the default configuration", "Error should mention default configuration")
}

func TestContext_Fill_AppName_FromDefault(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`app-name = "service-name"

[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "test_password",
		"APP_ENV":     "production",
	})

	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	require.NoError(t, err, "Fill() should not return error")
	assert.Equal(t, "service-name", config.AppName, "AppName should be loaded from default config")
	assert.Equal(t, "production", config.Environment, "Environment should be production")
}

func TestContext_Fill_AppName_Empty(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "test_password",
		"APP_ENV":     "development",
	})

	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	require.NoError(t, err, "Fill() should not return error")
	assert.Empty(t, config.AppName, "AppName should be empty when not specified in TOML")
	assert.Equal(t, "development", config.Environment, "Environment should be development")
}

func TestContext_Fill_AppName_OverrideInEnvConfig(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`app-name = "default-service"

[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
		"configs/production.toml": &fstest.MapFile{
			Data: []byte(`app-name = "production-service"`),
		},
	}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "test_password",
		"APP_ENV":     "production",
	})

	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	require.NoError(t, err, "Fill() should not return error")
	assert.Equal(t, "production-service", config.AppName, "AppName should be overridden by production config")
	assert.Equal(t, "production", config.Environment, "Environment should be production")
}

func TestContext_Fill_AppName_SpecialCharacters(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	mockFS := fstest.MapFS{
		"configs/default.toml": &fstest.MapFile{
			Data: []byte(`app-name = "my-app_v1.0"

[database]
user = "rcv"
host = "localhost"
port = "4332"
name = "poc"
`),
		},
	}

	setEnv(t, map[string]string{
		"DB_PASSWORD": "test_password",
		"APP_ENV":     "staging",
	})

	configCtx := NewContext(ctx, mockFS, "")
	config := &TestConfig{}
	err := configCtx.Fill(config)

	require.NoError(t, err, "Fill() should not return error")
	assert.Equal(t, "my-app_v1.0", config.AppName, "AppName should handle special characters")
	assert.Equal(t, "staging", config.Environment, "Environment should be staging")
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
		require.NoError(t, err, "envUnmarshal() should not return error with valid envs map")

		assert.Equal(t, "production", config.Environment, "Environment should be set from envs map")
		assert.Equal(t, "secret_pass", config.Database.Password, "Password should be set from envs map")
	})

	t.Run("with nil envs uses OS environment", func(t *testing.T) {
		config := &TestConfig{}

		// This will likely fail since OS env vars aren't set, but shouldn't panic
		err := envUnmarshal(ctx, config, nil, "")
		if err != nil {
			t.Logf("envUnmarshal() with nil envs failed as expected: %v", err)
		}
	})
}
