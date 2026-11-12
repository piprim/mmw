package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/sethvargo/go-envconfig"
)

const (
	// EnvAgeIdentity holds an age identity (the secret key text itself).
	EnvAgeIdentity = "AGE_IDENTITY"
	// EnvAgeIdentityFile holds the path of an age identity file.
	EnvAgeIdentityFile = "AGE_IDENTITY_FILE"
	// SystemIdentityFile is the last-resort identity file location.
	SystemIdentityFile = "/etc/age/keys.txt"
)

// ErrNoAgeIdentity is returned when no age identity can be found.
var ErrNoAgeIdentity = errors.New("no age identity found")

// defaultIdentityPaths lists the identity files to try, in order.
func defaultIdentityPaths(l envconfig.Lookuper) []string {
	var paths []string
	if xdg, ok := l.Lookup("XDG_CONFIG_HOME"); ok && xdg != "" {
		paths = append(paths, filepath.Join(xdg, "age", "keys.txt"))
	} else if home, ok := l.Lookup("HOME"); ok && home != "" {
		paths = append(paths, filepath.Join(home, ".config", "age", "keys.txt"))
	}

	return append(paths, SystemIdentityFile)
}

// loadIdentities returns the identities from AGE_IDENTITY, else AGE_IDENTITY_FILE,
// else the first existing file of paths.
func loadIdentities(l envconfig.Lookuper, paths []string) ([]age.Identity, error) {
	if text, ok := l.Lookup(EnvAgeIdentity); ok && text != "" {
		ids, err := age.ParseIdentities(strings.NewReader(text))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", EnvAgeIdentity, err)
		}

		return ids, nil
	}

	if path, ok := l.Lookup(EnvAgeIdentityFile); ok && path != "" {
		ids, err := parseIdentityFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", EnvAgeIdentityFile, err)
		}

		return ids, nil
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}

		return parseIdentityFile(path)
	}

	return nil, fmt.Errorf("%w: set %s or %s, or put an identity file at one of %s",
		ErrNoAgeIdentity, EnvAgeIdentity, EnvAgeIdentityFile, strings.Join(paths, ", "))
}

func parseIdentityFile(path string) ([]age.Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	ids, err := age.ParseIdentities(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return ids, nil
}

// decryptOverlay decrypts an age file, armored or binary. name only labels errors.
func decryptOverlay(name string, data []byte, ids []age.Identity) ([]byte, error) {
	var src io.Reader = bytes.NewReader(data)
	if bytes.HasPrefix(data, []byte(armor.Header)) {
		src = armor.NewReader(src)
	}

	r, err := age.Decrypt(src, ids...)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", name, err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", name, err)
	}

	return plain, nil
}
