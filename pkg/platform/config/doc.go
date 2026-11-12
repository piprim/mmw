// Package config provides unified configuration management for Go applications.
//
// It combines TOML configuration files with environment variables, supporting
// hierarchical configuration loading with environment-specific overrides.
//
// # Configuration Loading
//
// The package loads configuration in this order (later sources override earlier):
//
//  1. Load configs/default.toml (required)
//  2. Load configs/<APP_ENV>.toml, or decrypt configs/<APP_ENV>.toml.age with an age
//     identity (AGE_IDENTITY, AGE_IDENTITY_FILE, $XDG_CONFIG_HOME/age/keys.txt,
//     $HOME/.config/age/keys.txt, /etc/age/keys.txt). Optional for development and
//     testing, required (fail closed) for any other environment.
//  3. In development only, load configs/local.toml (optional)
//  4. Apply environment variables (override TOML when set)
//
// # Usage
//
// Define your configuration struct with appropriate tags:
//
//	type AppConfig struct {
//	    Port        string    `mapstructure:"port"`
//	    Environment string    `env:"APP_ENV, required"`
//	    Database    *DBConfig `mapstructure:"database"`
//	}
//
//	func (c AppConfig) GetAppEnv() string {
//	    return c.Environment
//	}
//
// Load configuration using an embedded filesystem:
//
//	//go:embed configs/*
//	var configFS embed.FS
//
//	ctx := context.Background()
//	cfg := &AppConfig{}
//	err := config.NewContext(ctx, configFS, "").Fill(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Testing
//
// For unit tests, use testing/fstest.MapFS to mock the filesystem:
//
//	mockFS := fstest.MapFS{
//	    "configs/default.toml": &fstest.MapFile{
//	        Data: []byte(`port = "8080"`),
//	    },
//	}
//	envs := map[string]string{"APP_ENV": "testing"}
//	configCtx := config.NewContext(ctx, mockFS, envs)
//
// # Struct Tags
//
//   - mapstructure: Maps TOML/JSON/YAML fields to struct fields (use kebab-case)
//   - env: Maps environment variables to struct fields
package config
