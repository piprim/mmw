package config

import (
	"context"
	"embed"
	"io/fs"

	pfconfig "{{.PlatformPath}}/pkg/platform/config"
	"github.com/rotisserie/eris"
)

//go:embed configs/*.toml
var embeddedFS embed.FS

var getConfigFS = func() fs.FS { return embeddedFS }

// Config holds {{.NameTitle}} module configuration.
type Config struct {
	pfconfig.Base
	Database *pfconfig.Database `mapstructure:"database"`
	Server   *pfconfig.Server   `mapstructure:"server"`
}

var conf *Config

// Load reads the TOML configuration files.
func Load(ctx context.Context, envPrefix string) (*Config, error) {
	if conf != nil {
		return conf, nil
	}

	conf := new(Config)
	if err := pfconfig.NewContext(ctx, getConfigFS(), envPrefix).Fill(conf); err != nil {
		return nil, eris.Wrap(err, "error filling {{.Name}} config")
	}

	return conf, nil
}
