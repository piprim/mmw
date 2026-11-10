package config

import (
	"context"
	"embed"
	"io/fs"

	pfconfig "[[.PlatformPath]]/pkg/platform/config"
	pfslog "[[.PlatformPath]]/pkg/platform/slog"
	"github.com/rotisserie/eris"
)

//go:embed configs/*
var embeddedFS embed.FS

// SecretPlaceholders are the fake values refused outside development and testing.
var SecretPlaceholders = []string{"rcv", "fake-secret", "fake-token", "CHANGE-ME"}

// getConfigFS returns the filesystem to use for reading config files.
// This is a variable so it can be mocked in tests.
var getConfigFS = func() fs.FS {
	return embeddedFS
}

// Gateway is the external provider the module's gateway slices send through.
// The token is a fake value in default.toml; the real one lives in
// development.toml.age or GATEWAY_TOKEN.
// ponytail: one URL and token per module; move to [gateway.<entity>] when a
// module talks to two providers.
type Gateway struct {
	URL   string `mapstructure:"url"`
	Token string `env:"GATEWAY_TOKEN" json:"-" mapstructure:"token"`
}

type Config struct {
	pfconfig.Base
[[- if .Database]]

	Database *pfconfig.Database `mapstructure:"database"`
[[- end]]

	Gateway *Gateway `mapstructure:"gateway"`
	// Environment pfconfig.Environment `env:"APP_ENV, required" mapstructure:"environment"`
[[- if .Grpc]]

	Server     *pfconfig.Server `mapstructure:"server"`
	AuthServer *pfconfig.Server `mapstructure:"auth-server"`
[[- end]]

	LogLevel pfslog.LogLevel `mapstructure:"log-level"`
}

// Load reads the TOML files and automatically overrides them with Env Vars
// Load loads the configurations from embedded files:
// - configs/default.toml
// - configs/<APP_ENV>.toml if exist
// If envs is not nil, automatically overrides them with Env Vars.
func Load(ctx context.Context, envPrefix string) (*Config, error) {
	conf := new(Config)
	configFS := getConfigFS()

	err := pfconfig.NewContext(ctx, configFS, envPrefix).Fill(conf)
	if err != nil {
		return nil, eris.Wrap(err, "error filling config")
	}

	var gatewayToken string
	if conf.Gateway != nil {
		gatewayToken = conf.Gateway.Token
	}

	if err := pfconfig.RequireSecret(conf.Environment, "gateway token", gatewayToken, SecretPlaceholders...); err != nil {
		return nil, eris.Wrap(err, "invalid configuration")
	}

	return conf, nil
}
