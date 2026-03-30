package config

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"

	oglos "github.com/piprim/mmw/pkg/platform/os"
	"github.com/sethvargo/go-envconfig"
	"github.com/spf13/viper"
)

// Context is the context needed to fill a configuration
type Context struct {
	ctx    context.Context
	fs     fs.FS
	envs   map[string]string
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
	envs := oglos.EnvMap()
	if err := envUnmarshal(c.ctx, config, envs, c.prefix); err != nil {
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

	file := config.GetAppEnv().String() + ".toml"
	envConfig, err := fs.ReadFile(configFS, "configs/"+file)
	if err == nil {
		if err := v.MergeConfig(bytes.NewBuffer(envConfig)); err != nil {
			return fmt.Errorf("viper failed merging the configuration %s: %w", file, err)
		}
	}

	if err := envUnmarshal(c.ctx, config, c.envs, c.prefix); err != nil {
		return err
	}

	if err := v.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

func envUnmarshal(ctx context.Context, config Config, envs map[string]string, prefix string) error {
	var baseLookuper envconfig.Lookuper
	if envs == nil {
		baseLookuper = envconfig.OsLookuper()
	} else {
		baseLookuper = envconfig.MapLookuper(envs)
	}

	lookuper := baseLookuper

	if prefix != "" {
		lookuper = envconfig.MultiLookuper(
			envconfig.PrefixLookuper(prefix, baseLookuper),
			baseLookuper,
		)
	}

	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:   config,
		Lookuper: lookuper,
	}); err != nil {
		return fmt.Errorf("envconfig process error: %w", err)
	}

	return nil
}
