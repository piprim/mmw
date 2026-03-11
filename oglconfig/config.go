package oglconfig

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"

	"github.com/sethvargo/go-envconfig"
	"github.com/spf13/viper"
)

// Context is the context needed to fill a configuration
type Context struct {
	// Context used to fill the configuration
	ctx context.Context
	// - Must contain the default configuration file: configs/default.toml
	// - Optionally a specific APP_ENV config file: configs/<APP_ENV>.toml
	fs fs.FS
	// If envs is not nil, use as environnement variable (eg. mocking)
	envs map[string]string
}

// Config is the interface that configuration structs must implement.
// GetAppEnv returns the environment name (e.g., "production", "development")
// which is used to load environment-specific configuration files.
type Config interface {
	GetAppEnv() fmt.Stringer
}

// NewContext creates a new configuration context.
// The fs parameter must contain configs/default.toml and optionally configs/<APP_ENV>.toml.
// If envs is nil, environment variables are read from the OS; otherwise the provided map is used.
func NewContext(ctx context.Context, lfs fs.FS, envs map[string]string) *Context {
	return &Context{
		ctx:  ctx,
		fs:   lfs,
		envs: envs,
	}
}

// Fill fills the configuration `config` from the context `c`.
func (c *Context) Fill(config Config) error {
	if err := envUnmarshal(c.ctx, config, c.envs); err != nil {
		return err
	}

	viper.SetConfigType("toml")
	configFS := c.fs
	defaultConfig, err := fs.ReadFile(configFS, "configs/default.toml")
	if err != nil {
		return fmt.Errorf("failed to read the default configuration: %w", err)
	}
	if err := viper.ReadConfig(bytes.NewBuffer(defaultConfig)); err != nil {
		return fmt.Errorf("viper failed to read the default configuration: %w", err)
	}

	file := config.GetAppEnv().String() + ".toml"
	envConfig, err := fs.ReadFile(configFS, "configs/"+file)
	if err == nil { // Env config may not exist.
		// Merge environment-specific config
		if err := viper.MergeConfig(bytes.NewBuffer(envConfig)); err != nil {
			return fmt.Errorf("viper failed merging the configuration %s: %w", file, err)
		}
	}

	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

func envUnmarshal(ctx context.Context, config Config, envs map[string]string) error {
	if envs == nil {
		if err := envconfig.Process(ctx, config); err != nil {
			return fmt.Errorf("envconfig process error: %w", err)
		}

		return nil
	}

	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:   config,
		Lookuper: envconfig.MapLookuper(envs),
	}); err != nil {
		// Do not expose `envs` because of secrets!
		return fmt.Errorf("envconfig process error: %w", err)
	}

	return nil
}
