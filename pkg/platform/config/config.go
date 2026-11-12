package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/sethvargo/go-envconfig"
	"github.com/spf13/viper"
)

// LocalOverlay is the optional developer-local file merged last in development.
const LocalOverlay = "configs/local.toml"

var (
	// ErrOverlayRequired is returned when a non-development environment has no overlay.
	ErrOverlayRequired = errors.New("environment overlay is required")
	// ErrOverlayAmbiguous is returned when both <env>.toml and <env>.toml.age exist.
	ErrOverlayAmbiguous = errors.New("both <env>.toml and <env>.toml.age exist")
)

// Context is the context needed to fill a configuration
type Context struct {
	ctx    context.Context
	fs     fs.FS
	prefix string
}

// Config is the interface that configuration structs must implement.
// GetAppEnv returns the environment name (e.g., "production", "development")
// which is used to load environment-specific configuration files.
type Config interface {
	// GetAppEnv return the app environment: development, production, staging, etc
	GetAppEnv() fmt.Stringer
}

// NewContext creates a new configuration context.
// The fs parameter must contain configs/default.toml and optionally configs/<APP_ENV>.toml.
// If envs is nil, environment variables are read from the OS; otherwise the provided map is used.
func NewContext(ctx context.Context, lfs fs.FS, envprefix string) *Context {
	return &Context{
		ctx:    ctx,
		fs:     lfs,
		prefix: envprefix,
	}
}

// Fill fills the configuration `config` from the context `c`.
func (c *Context) Fill(config Config) error {
	if err := envUnmarshal(c.ctx, config, c.lookuper(), false); err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigType("toml")
	configFS := c.fs
	defaultConfig, err := fs.ReadFile(configFS, "configs/default.toml")
	if err != nil {
		return fmt.Errorf("failed to read the default configuration: %w", err)
	}
	if err := v.ReadConfig(bytes.NewBuffer(defaultConfig)); err != nil {
		return fmt.Errorf("viper failed to read the default configuration: %w", err)
	}

	env := config.GetAppEnv().String()
	overlay, err := c.readOverlay(env)
	if err != nil {
		return err
	}
	if overlay != nil {
		name := env + ".toml"
		if _, err := fs.Stat(c.fs, "configs/"+name+".age"); err == nil {
			name += ".age"
		}
		if err := v.MergeConfig(bytes.NewReader(overlay)); err != nil {
			return fmt.Errorf("viper failed merging the configuration %s: %w", name, err)
		}
	}

	if env == EnvironmentDevelopment.String() {
		if local, err := fs.ReadFile(c.fs, LocalOverlay); err == nil {
			if err := v.MergeConfig(bytes.NewReader(local)); err != nil {
				return fmt.Errorf("viper failed merging %s: %w", LocalOverlay, err)
			}
		}
	}

	if err := v.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Env vars win over TOML when set.
	if err := envUnmarshal(c.ctx, config, c.lookuper(), true); err != nil {
		return err
	}

	return nil
}

// readOverlay returns the <env> overlay: clear, or decrypted from <env>.toml.age.
// It returns (nil, nil) when none exists and env is development or testing.
func (c *Context) readOverlay(env string) ([]byte, error) {
	clearPath := "configs/" + env + ".toml"
	encrypted := clearPath + ".age"
	_, clearErr := fs.Stat(c.fs, clearPath)
	_, encErr := fs.Stat(c.fs, encrypted)

	switch {
	case clearErr == nil && encErr == nil:
		return nil, fmt.Errorf("%w: %s and %s", ErrOverlayAmbiguous, clearPath, encrypted)
	case clearErr == nil:
		data, err := fs.ReadFile(c.fs, clearPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", clearPath, err)
		}

		return data, nil
	case encErr == nil:
		data, err := fs.ReadFile(c.fs, encrypted)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", encrypted, err)
		}
		l := c.lookuper()
		ids, err := loadIdentities(l, defaultIdentityPaths(l))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", encrypted, err)
		}

		return decryptOverlay(encrypted, data, ids)
	case env == EnvironmentDevelopment.String() || env == EnvironmentTesting.String():
		return nil, nil
	}

	return nil, fmt.Errorf("%w: add %s or %s", ErrOverlayRequired, clearPath, encrypted)
}

// lookuper reads the OS environment, trying the prefixed name first when a prefix is set.
func (c *Context) lookuper() envconfig.Lookuper {
	base := envconfig.OsLookuper()
	if c.prefix == "" {
		return base
	}

	return envconfig.MultiLookuper(envconfig.PrefixLookuper(c.prefix, base), base)
}

func envUnmarshal(ctx context.Context, config Config, l envconfig.Lookuper, overwrite bool) error {
	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:           config,
		Lookuper:         l,
		DefaultOverwrite: overwrite,
	}); err != nil {
		return fmt.Errorf("envconfig process error: %w", err)
	}

	return nil
}
