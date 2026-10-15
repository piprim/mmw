[[if .Database -]]
package config

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	pfconfig "[[.PlatformPath]]/pkg/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDefaultTOML = `
[database]
user = "rcv"
host = "localhost"
port = 5432
name = "testdb"
`

func withConfigFS(t *testing.T, files fstest.MapFS) {
	t.Helper()
	orig := getConfigFS
	getConfigFS = func() fs.FS { return files }
	t.Cleanup(func() { getConfigFS = orig })
}

func TestLoad(t *testing.T) {
	t.Run("merges default and APP_ENV files and reads the password from the environment", func(t *testing.T) {
		withConfigFS(t, fstest.MapFS{
			"configs/default.toml": {Data: []byte(testDefaultTOML)},
			"configs/testing.toml": {Data: []byte("[database]\nhost = \"test-host\"\n")},
		})
		t.Setenv("APP_ENV", "testing")
		t.Setenv("DB_PASSWORD", "secret123")

		cfg, err := Load(context.Background(), "")
		require.NoError(t, err)
		assert.Equal(t, pfconfig.EnvironmentTesting, cfg.Environment)
		assert.Equal(t, "test-host", cfg.Database.Host)
		assert.Contains(t, cfg.Database.URL(), "secret123")
	})

	t.Run("fails when the database password is empty", func(t *testing.T) {
		withConfigFS(t, fstest.MapFS{"configs/default.toml": {Data: []byte(testDefaultTOML)}})
		t.Setenv("APP_ENV", "development")
		t.Setenv("DB_PASSWORD", "")

		cfg, err := Load(context.Background(), "")
		assert.Nil(t, cfg)
		assert.ErrorContains(t, err, "database password is empty")
	})
}
[[- else -]]
package config

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	pfconfig "[[.PlatformPath]]/pkg/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDefaultTOML = `
[gateway]
url = "http://localhost:9999/[[.Entity]]"
`

func withConfigFS(t *testing.T, files fstest.MapFS) {
	t.Helper()
	orig := getConfigFS
	getConfigFS = func() fs.FS { return files }
	t.Cleanup(func() { getConfigFS = orig })
}

func TestLoad(t *testing.T) {
	t.Run("merges default and APP_ENV files and reads the token from the environment", func(t *testing.T) {
		withConfigFS(t, fstest.MapFS{
			"configs/default.toml": {Data: []byte(testDefaultTOML)},
			"configs/testing.toml": {Data: []byte("[gateway]\nurl = \"http://test-host/[[.Entity]]\"\n")},
		})
		t.Setenv("APP_ENV", "testing")
		t.Setenv("GATEWAY_TOKEN", "secret123")

		cfg, err := Load(context.Background(), "")
		require.NoError(t, err)
		assert.Equal(t, pfconfig.EnvironmentTesting, cfg.Environment)
		assert.Equal(t, "http://test-host/[[.Entity]]", cfg.Gateway.URL)
		assert.Equal(t, "secret123", cfg.Gateway.Token)
	})

	t.Run("fails when the gateway url is empty", func(t *testing.T) {
		withConfigFS(t, fstest.MapFS{"configs/default.toml": {Data: []byte("log-level=\"debug\"\n")}})
		t.Setenv("APP_ENV", "development")

		cfg, err := Load(context.Background(), "")
		assert.Nil(t, cfg)
		assert.ErrorContains(t, err, "gateway url is empty")
	})
}
[[- end]]
