package config

import (
	"fmt"
	"os"
)

type Base struct {
	Environment Environment `env:"APP_ENV, required" mapstructure:"environment"`
}

func (b *Base) GetAppEnv() fmt.Stringer {
	if os.Getenv("APP_ENV") == "" { // APP_ENV is the key point of the configuration process
		panic("environnement varibale APP_ENV is not set")
	}

	return b.Environment
}
