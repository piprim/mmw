package config

import (
	"context"
	"embed"
	"io/fs"

	pfconfig "[[.PlatformPath]]/pkg/platform/config"
	pfslog "[[.PlatformPath]]/pkg/platform/slog"
	"github.com/rotisserie/eris"
)

//go:embed configs/*.toml
var embeddedFS embed.FS

// getConfigFS returns the filesystem to use for reading config files.
// This is a variable so it can be mocked in tests.
var getConfigFS = func() fs.FS {
	return embeddedFS
}

[[- if not .Database]]

// Gateway is the external provider this module sends through. The token
// comes from the environment only, like a database password.
type Gateway struct {
	URL   string `mapstructure:"url"`
	Token string `env:"GATEWAY_TOKEN" json:"-" mapstructure:"-"`
}
[[- end]]

type Config struct {
	pfconfig.Base
[[- if .Database]]

	Database *pfconfig.Database `mapstructure:"database"`
[[- else]]

	Gateway *Gateway `mapstructure:"gateway"`
[[- end]]
	// Environment pfconfig.Environment `env:"APP_ENV, required" mapstructure:"environment"`
[[- if .Grpc]]

	Server     *pfconfig.Server `mapstructure:"server"`
	AuthServer *pfconfig.Server `mapstructure:"auth-server"`
[[- end]]

	LogLevel pfslog.LogLevel `mapstructure:"log-level"`
}

var conf *Config

// Load reads the TOML files and automatically overrides them with Env Vars
// Load loads the configurations from embedded files:
// - configs/default.toml
// - configs/<APP_ENV>.toml if exist
// If envs is not nil, automatically overrides them with Env Vars.
func Load(ctx context.Context, envPrefix string) (*Config, error) {
	if conf != nil {
		return conf, nil
	}

	conf := new(Config)
	configFS := getConfigFS()

	err := pfconfig.NewContext(ctx, configFS, envPrefix).Fill(conf)
	if err != nil {
		return nil, eris.Wrap(err, "error filling config")
	}

[[- if .Database]]
	if conf.Database.Password == "" {
		return nil, eris.New("database password is empty")
	}
[[- else]]
	if conf.Gateway == nil || conf.Gateway.URL == "" {
		return nil, eris.New("gateway url is empty")
	}
[[- end]]

	return conf, nil
}
