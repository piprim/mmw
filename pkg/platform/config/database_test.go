package config

import (
	"context"
	"fmt"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabase_URL(t *testing.T) {
	t.Run("uses postgres scheme by default", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb"}
		assert.Contains(t, d.URL(), "postgres://")
	})

	t.Run("uses custom scheme when specified", func(t *testing.T) {
		d := &Database{Scheme: "postgresql", Host: "localhost", Port: 5432, Name: "mydb"}
		assert.Contains(t, d.URL(), "postgresql://")
	})

	t.Run("includes user without colon when no password", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb", User: "admin"}
		url := d.URL()
		assert.Contains(t, url, "admin@")
		assert.NotContains(t, url, ":@")
	})

	t.Run("includes user:password when both set", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb", User: "admin", Password: "secret"}
		assert.Contains(t, d.URL(), "admin:secret@")
	})

	t.Run("appends sslmode query param when set", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb", SSLMode: "require"}
		assert.Contains(t, d.URL(), "sslmode=require")
	})

	t.Run("omits sslmode query param when empty", func(t *testing.T) {
		d := &Database{Host: "localhost", Port: 5432, Name: "mydb"}
		assert.NotContains(t, d.URL(), "sslmode")
	})

	t.Run("full URL with all fields", func(t *testing.T) {
		d := &Database{
			Scheme:   "postgres",
			Host:     "db.example.com",
			Port:     5432,
			Name:     "appdb",
			User:     "user",
			Password: "pass",
			SSLMode:  "disable",
		}
		url := d.URL()
		assert.Contains(t, url, "postgres://")
		assert.Contains(t, url, "user:pass@")
		assert.Contains(t, url, "db.example.com:5432")
		assert.Contains(t, url, "/appdb")
		assert.Contains(t, url, "sslmode=disable")
	})
}

type realDBConfig struct {
	Environment string    `env:"APP_ENV, required"`
	Database    *Database `mapstructure:"database"`
}

func (c *realDBConfig) GetAppEnv() fmt.Stringer { return envString(c.Environment) }

func TestDatabase_PasswordPrecedence(t *testing.T) {
	fill := func(t *testing.T, env map[string]string) *realDBConfig {
		t.Helper()
		resetViper()
		t.Cleanup(resetViper)
		env["APP_ENV"] = "development"
		setEnv(t, env)
		if _, ok := env["DB_PASSWORD"]; !ok {
			os.Unsetenv("DB_PASSWORD")
		}
		mockFS := fstest.MapFS{"configs/default.toml": &fstest.MapFile{Data: []byte("[database]\nuser = \"rcv\"\npassword = \"from-toml\"\n")}}
		cfg := &realDBConfig{}
		require.NoError(t, NewContext(context.Background(), mockFS, "").Fill(cfg))
		require.NotNil(t, cfg.Database)

		return cfg
	}

	t.Run("password read from TOML when DB_PASSWORD unset", func(t *testing.T) {
		assert.Equal(t, "from-toml", fill(t, map[string]string{}).Database.Password)
	})

	t.Run("DB_PASSWORD overrides TOML password", func(t *testing.T) {
		assert.Equal(t, "from-env", fill(t, map[string]string{"DB_PASSWORD": "from-env"}).Database.Password)
	})
}
