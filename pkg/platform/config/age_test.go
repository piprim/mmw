package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// encryptArmored encrypts plaintext for r and returns the ASCII-armored ciphertext.
func encryptArmored(t *testing.T, plaintext string, r age.Recipient) []byte {
	t.Helper()

	var buf bytes.Buffer
	aw := armor.NewWriter(&buf)
	w, err := age.Encrypt(aw, r)
	require.NoError(t, err)
	_, err = w.Write([]byte(plaintext))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, aw.Close())

	return buf.Bytes()
}

func newIdentity(t *testing.T) *age.X25519Identity {
	t.Helper()
	id, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	return id
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestDefaultIdentityPaths(t *testing.T) {
	t.Run("XDG_CONFIG_HOME set", func(t *testing.T) {
		got := defaultIdentityPaths(envconfig.MapLookuper(map[string]string{"XDG_CONFIG_HOME": "/x"}))
		assert.Equal(t, []string{"/x/age/keys.txt", "/etc/age/keys.txt"}, got)
	})

	t.Run("HOME only", func(t *testing.T) {
		got := defaultIdentityPaths(envconfig.MapLookuper(map[string]string{"HOME": "/h"}))
		assert.Equal(t, []string{"/h/.config/age/keys.txt", "/etc/age/keys.txt"}, got)
	})

	t.Run("neither", func(t *testing.T) {
		got := defaultIdentityPaths(envconfig.MapLookuper(map[string]string{}))
		assert.Equal(t, []string{"/etc/age/keys.txt"}, got)
	})
}

func TestLoadIdentities(t *testing.T) {
	id := newIdentity(t)

	t.Run("AGE_IDENTITY text", func(t *testing.T) {
		ids, err := loadIdentities(envconfig.MapLookuper(map[string]string{EnvAgeIdentity: id.String()}), nil)
		require.NoError(t, err)
		assert.Len(t, ids, 1)
	})

	t.Run("AGE_IDENTITY_FILE path", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "id.txt")
		writeFile(t, p, "# comment\n"+id.String()+"\n")
		ids, err := loadIdentities(envconfig.MapLookuper(map[string]string{EnvAgeIdentityFile: p}), nil)
		require.NoError(t, err)
		assert.Len(t, ids, 1)
	})

	t.Run("AGE_IDENTITY_FILE set but unreadable is an error", func(t *testing.T) {
		_, err := loadIdentities(envconfig.MapLookuper(map[string]string{EnvAgeIdentityFile: "/nonexistent"}), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), EnvAgeIdentityFile)
	})

	t.Run("AGE_IDENTITY wins over the default file", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "keys.txt")
		writeFile(t, p, newIdentity(t).String()+"\n")
		ids, err := loadIdentities(envconfig.MapLookuper(map[string]string{EnvAgeIdentity: id.String()}), []string{p})
		require.NoError(t, err)
		require.Len(t, ids, 1)
		assert.Equal(t, id.Recipient().String(), ids[0].(*age.X25519Identity).Recipient().String())
	})

	t.Run("default path used when no variable is set", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "keys.txt")
		writeFile(t, p, id.String()+"\n")
		ids, err := loadIdentities(envconfig.MapLookuper(nil), []string{p})
		require.NoError(t, err)
		assert.Len(t, ids, 1)
	})

	t.Run("second default path used when the first is absent", func(t *testing.T) {
		dir := t.TempDir()
		present := filepath.Join(dir, "keys.txt")
		writeFile(t, present, id.String()+"\n")
		ids, err := loadIdentities(envconfig.MapLookuper(nil), []string{filepath.Join(dir, "missing"), present})
		require.NoError(t, err)
		assert.Len(t, ids, 1)
	})

	t.Run("nothing found", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing")
		_, err := loadIdentities(envconfig.MapLookuper(nil), []string{missing})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNoAgeIdentity))
		assert.Contains(t, err.Error(), missing)
	})

	t.Run("file with several identities", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "keys.txt")
		writeFile(t, p, id.String()+"\n"+newIdentity(t).String()+"\n")
		ids, err := loadIdentities(envconfig.MapLookuper(nil), []string{p})
		require.NoError(t, err)
		assert.Len(t, ids, 2)
	})
}

func TestDecryptOverlay(t *testing.T) {
	const plain = "[database]\npassword = \"s3cret\"\n"
	id := newIdentity(t)

	t.Run("armored round trip", func(t *testing.T) {
		got, err := decryptOverlay("production.toml.age", encryptArmored(t, plain, id.Recipient()), []age.Identity{id})
		require.NoError(t, err)
		assert.Equal(t, plain, string(got))
	})

	t.Run("wrong identity", func(t *testing.T) {
		_, err := decryptOverlay("production.toml.age", encryptArmored(t, plain, id.Recipient()), []age.Identity{newIdentity(t)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decrypt production.toml.age")
	})

	t.Run("truncated armor", func(t *testing.T) {
		data := encryptArmored(t, plain, id.Recipient())
		_, err := decryptOverlay("production.toml.age", data[:len(data)-40], []age.Identity{id})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "production.toml.age")
	})

	t.Run("unarmored binary round trip", func(t *testing.T) {
		var buf bytes.Buffer
		w, err := age.Encrypt(&buf, id.Recipient())
		require.NoError(t, err)
		_, err = w.Write([]byte(plain))
		require.NoError(t, err)
		require.NoError(t, w.Close())

		got, err := decryptOverlay("production.toml.age", buf.Bytes(), []age.Identity{id})
		require.NoError(t, err)
		assert.Equal(t, plain, string(got))
	})
}
