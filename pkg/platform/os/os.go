package os

import "os"

// EnvMap returns the current process environment as a map[string]string.
// Each entry is expected to be in "KEY=VALUE" format; entries without "="
// are stored with an empty string value.
func EnvMap() map[string]string {
	env := os.Environ()
	m := make(map[string]string, len(env))
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				m[e[:i]] = e[i+1:]

				break
			}
		}
	}

	return m
}
