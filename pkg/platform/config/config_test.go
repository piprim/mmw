package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/sethvargo/go-envconfig"
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
	Password string `env:"DB_PASSWORD"`
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

	})

	pwFS := func() fstest.MapFS {
		return fstest.MapFS{"configs/default.toml": &fstest.MapFile{Data: []byte("[database]\nuser = \"rcv\"\npassword = \"from-toml\"\n")}}
	}
	fillPw := func(t *testing.T, env map[string]string) string {
		t.Helper()
		resetViper()
		t.Cleanup(resetViper)
		env["APP_ENV"] = "development"
		setEnv(t, env)
		if _, ok := env["DB_PASSWORD"]; !ok {
			os.Unsetenv("DB_PASSWORD")
		}
		config := &TestConfig{}
		require.NoError(t, NewContext(context.Background(), pwFS(), "").Fill(config))
		return config.Database.Password
	}

	t.Run("TOML provides the password when the env var is unset", func(t *testing.T) {
		assert.Equal(t, "from-toml", fillPw(t, map[string]string{}))
	})
	t.Run("env var overrides the TOML password", func(t *testing.T) {
		assert.Equal(t, "from-env", fillPw(t, map[string]string{"DB_PASSWORD": "from-env"}))
	})
	t.Run("env var set but empty keeps the TOML password", func(t *testing.T) {
		assert.Equal(t, "from-toml", fillPw(t, map[string]string{"DB_PASSWORD": ""}))
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

		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "testing"})

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

		setEnv(t, map[string]string{"DB_PASSWORD": "os_test_password", "APP_ENV": "testing"})

		config := &TestConfig{}
		err := NewContext(context.Background(), defaultFS(), "").Fill(config)

		require.NoError(t, err)
		require.NotNil(t, config.Database)
		assert.Equal(t, "os_test_password", config.Database.Password)
		assert.Equal(t, "testing", config.Environment)
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

		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "testing"})

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
		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "development"})

		config := &TestConfig{}
		require.NoError(t, NewContext(context.Background(), mockFS, "").Fill(config))
		assert.Equal(t, "service-name", config.AppName)
		assert.Equal(t, "development", config.Environment)
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
		setEnv(t, map[string]string{"DB_PASSWORD": "test_password", "APP_ENV": "development"})

		config := &TestConfig{}
		require.NoError(t, NewContext(context.Background(), mockFS, "").Fill(config))
		assert.Equal(t, "my-app_v1.0", config.AppName)
		assert.Equal(t, "development", config.Environment)
	})
}

func TestEnvUnmarshal(t *testing.T) {
	resetViper()
	defer resetViper()
	ctx := context.Background()

	t.Run("reads the OS environment", func(t *testing.T) {
		setEnv(t, map[string]string{
			"DB_PASSWORD": "secret_pass",
			"APP_ENV":     "production",
		})

		config := &TestConfig{}
		err := envUnmarshal(ctx, config, envconfig.OsLookuper(), false)
		require.NoError(t, err)

		assert.Equal(t, "production", config.Environment)
		assert.Equal(t, "secret_pass", config.Database.Password)
	})

	t.Run("missing required APP_ENV errors", func(t *testing.T) {
		setEnv(t, map[string]string{})
		os.Unsetenv("APP_ENV")
		assert.Error(t, envUnmarshal(ctx, &TestConfig{}, envconfig.OsLookuper(), false))
	})
}

func TestContext_Fill_Overlays(t *testing.T) {
	const defaultTOML = "[database]\nuser = \"rcv\"\n"
	const prodTOML = "[database]\npassword = \"prod-secret\"\nhost = \"db.internal\"\n"

	if _, err := os.Stat(SystemIdentityFile); err == nil {
		t.Logf("%s exists: the no-identity cases are skipped", SystemIdentityFile)
	}

	// env isolates the identity lookup from the machine and sets extra variables.
	env := func(t *testing.T, extra map[string]string) string {
		t.Helper()
		home := t.TempDir()
		m := map[string]string{"HOME": home}
		for k, v := range extra {
			m[k] = v
		}
		setEnv(t, m)
		for _, k := range []string{"XDG_CONFIG_HOME", "AGE_IDENTITY", "AGE_IDENTITY_FILE", "AUTH_AGE_IDENTITY"} {
			if _, set := m[k]; !set {
				t.Setenv(k, "")
				os.Unsetenv(k)
			}
		}

		return home
	}
	files := func(kv map[string]string) fstest.MapFS {
		fsys := fstest.MapFS{"configs/default.toml": &fstest.MapFile{Data: []byte(defaultTOML)}}
		for k, v := range kv {
			fsys["configs/"+k] = &fstest.MapFile{Data: []byte(v)}
		}

		return fsys
	}
	fill := func(t *testing.T, fsys fstest.MapFS, prefix string) (*TestConfig, error) {
		t.Helper()
		resetViper()
		t.Cleanup(resetViper)
		cfg := &TestConfig{}
		err := NewContext(context.Background(), fsys, prefix).Fill(cfg)

		return cfg, err
	}
	skipIfSystemKey := func(t *testing.T) {
		t.Helper()
		if _, err := os.Stat(SystemIdentityFile); err == nil {
			t.Skip(SystemIdentityFile + " exists on this machine")
		}
	}

	t.Run("encrypted overlay merged with AGE_IDENTITY", func(t *testing.T) {
		id := newIdentity(t)
		env(t, map[string]string{"APP_ENV": "production", "AGE_IDENTITY": id.String()})
		cfg, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, prodTOML, id.Recipient())), "")
		require.NoError(t, err)
		assert.Equal(t, "prod-secret", cfg.Database.Password)
		assert.Equal(t, "db.internal", cfg.Database.Host)
	})

	t.Run("encrypted overlay merged with AGE_IDENTITY_FILE", func(t *testing.T) {
		id := newIdentity(t)
		p := filepath.Join(t.TempDir(), "id.txt")
		writeFile(t, p, id.String()+"\n")
		env(t, map[string]string{"APP_ENV": "production", "AGE_IDENTITY_FILE": p})
		cfg, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, prodTOML, id.Recipient())), "")
		require.NoError(t, err)
		assert.Equal(t, "prod-secret", cfg.Database.Password)
		assert.Equal(t, "db.internal", cfg.Database.Host)
	})

	t.Run("encrypted overlay merged with the default identity file", func(t *testing.T) {
		id := newIdentity(t)
		home := env(t, map[string]string{"APP_ENV": "production"})
		writeFile(t, filepath.Join(home, ".config", "age", "keys.txt"), id.String()+"\n")
		cfg, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, prodTOML, id.Recipient())), "")
		require.NoError(t, err)
		assert.Equal(t, "prod-secret", cfg.Database.Password)
	})

	t.Run("a file holding two identities decrypts when the second one matches", func(t *testing.T) {
		id := newIdentity(t)
		home := env(t, map[string]string{"APP_ENV": "production"})
		writeFile(t, filepath.Join(home, ".config", "age", "keys.txt"), newIdentity(t).String()+"\n"+id.String()+"\n")
		cfg, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, prodTOML, id.Recipient())), "")
		require.NoError(t, err)
		assert.Equal(t, "prod-secret", cfg.Database.Password)
	})

	t.Run("prefixed identity variable is honoured", func(t *testing.T) {
		id := newIdentity(t)
		env(t, map[string]string{"APP_ENV": "production", "AUTH_AGE_IDENTITY": id.String()})
		cfg, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, prodTOML, id.Recipient())), "AUTH_")
		require.NoError(t, err)
		assert.Equal(t, "prod-secret", cfg.Database.Password)
	})

	t.Run("missing identity", func(t *testing.T) {
		skipIfSystemKey(t)
		env(t, map[string]string{"APP_ENV": "production"})
		_, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, prodTOML, newIdentity(t).Recipient())), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNoAgeIdentity))
		assert.Contains(t, err.Error(), ".config/age/keys.txt")
	})

	t.Run("present development.toml.age with no identity is an error", func(t *testing.T) {
		skipIfSystemKey(t)
		env(t, map[string]string{"APP_ENV": "development"})
		_, err := fill(t, fstestWith(files(nil), "development.toml.age", encryptArmored(t, prodTOML, newIdentity(t).Recipient())), "")
		assert.True(t, errors.Is(err, ErrNoAgeIdentity))
	})

	t.Run("encrypted development overlay overrides default.toml", func(t *testing.T) {
		id := newIdentity(t)
		env(t, map[string]string{"APP_ENV": "development", "AGE_IDENTITY": id.String()})
		fsys := fstestWith(files(nil), "default.toml", []byte("[database]\npassword = \"rcv\"\n"))
		fsys = fstestWith(fsys, "development.toml.age", encryptArmored(t, "[database]\npassword = \"real-dev\"\n", id.Recipient()))
		cfg, err := fill(t, fsys, "")
		require.NoError(t, err)
		assert.Equal(t, "real-dev", cfg.Database.Password)
	})

	t.Run("wrong identity", func(t *testing.T) {
		env(t, map[string]string{"APP_ENV": "production", "AGE_IDENTITY": newIdentity(t).String()})
		_, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, prodTOML, newIdentity(t).Recipient())), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "production.toml.age")
	})

	t.Run("overlay plaintext is not TOML", func(t *testing.T) {
		id := newIdentity(t)
		env(t, map[string]string{"APP_ENV": "production", "AGE_IDENTITY": id.String()})
		_, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, "this is not toml", id.Recipient())), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "production.toml.age")
	})

	t.Run("both clear and encrypted overlay", func(t *testing.T) {
		env(t, map[string]string{"APP_ENV": "staging"})
		fsys := fstestWith(files(map[string]string{"staging.toml": prodTOML}), "staging.toml.age", []byte("x"))
		_, err := fill(t, fsys, "")
		assert.True(t, errors.Is(err, ErrOverlayAmbiguous))
	})

	for _, e := range []string{"staging", "production"} {
		t.Run(e+" without overlay fails closed", func(t *testing.T) {
			env(t, map[string]string{"APP_ENV": e})
			_, err := fill(t, files(nil), "")
			assert.True(t, errors.Is(err, ErrOverlayRequired))
		})
	}

	for _, e := range []string{"development", "testing"} {
		t.Run(e+" without overlay is fine", func(t *testing.T) {
			env(t, map[string]string{"APP_ENV": e})
			_, err := fill(t, files(nil), "")
			assert.NoError(t, err)
		})
	}

	const localTOML = "[database]\nhost = \"local\"\nport = \"5435\"\n"

	t.Run("local.toml merged last in development", func(t *testing.T) {
		env(t, map[string]string{"APP_ENV": "development"})
		cfg, err := fill(t, files(map[string]string{"development.toml": "[database]\nhost = \"dev\"\n", "local.toml": localTOML}), "")
		require.NoError(t, err)
		assert.Equal(t, "local", cfg.Database.Host)
		assert.Equal(t, "5435", cfg.Database.Port)
	})

	t.Run("local.toml merged in development even without an overlay", func(t *testing.T) {
		env(t, map[string]string{"APP_ENV": "development"})
		cfg, err := fill(t, files(map[string]string{"local.toml": localTOML}), "")
		require.NoError(t, err)
		assert.Equal(t, "local", cfg.Database.Host)
	})

	t.Run("local.toml ignored in staging", func(t *testing.T) {
		env(t, map[string]string{"APP_ENV": "staging"})
		cfg, err := fill(t, files(map[string]string{"staging.toml": "[database]\nhost = \"stg\"\n", "local.toml": localTOML}), "")
		require.NoError(t, err)
		assert.Equal(t, "stg", cfg.Database.Host)
	})

	t.Run("env var still overrides the decrypted value", func(t *testing.T) {
		id := newIdentity(t)
		env(t, map[string]string{"APP_ENV": "production", "AGE_IDENTITY": id.String(), "DB_PASSWORD": "from-env"})
		cfg, err := fill(t, fstestWith(files(nil), "production.toml.age", encryptArmored(t, prodTOML, id.Recipient())), "")
		require.NoError(t, err)
		assert.Equal(t, "from-env", cfg.Database.Password)
	})
}

// fstestWith returns a copy of fsys with configs/<name> added or replaced.
func fstestWith[D string | []byte](fsys fstest.MapFS, name string, data D) fstest.MapFS {
	out := fstest.MapFS{}
	for k, v := range fsys {
		out[k] = v
	}
	out["configs/"+name] = &fstest.MapFile{Data: []byte(data)}

	return out
}
