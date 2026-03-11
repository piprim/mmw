package oglos

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

	environment := getenvironment(os.Environ(), func(item string) (key, val string) {
		splits := strings.SplitN(item, "=", 2)
		key = splits[0]
		val = splits[1]

		return
	})

	return environment
}
