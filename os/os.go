package os

import (
	"os"
	"strings"
)

// EnvMap return the environment varibales as a map, not a slice like `os.Environment`.
func EnvMap() map[string]string {
	getenvironment := func(data []string, getkeyval func(item string) (key, val string)) map[string]string {
		items := make(map[string]string)
		for _, item := range data {
			key, val := getkeyval(item)
			items[key] = val
		}

		return items
	}

	environment := getenvironment(os.Environ(), func(item string) (string, string) {
		splits := strings.SplitN(item, "=", 2)

		return splits[0], splits[1]
	})

	return environment
}
