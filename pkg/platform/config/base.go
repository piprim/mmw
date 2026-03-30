package config

import "fmt"

type Base struct {
	Environment Environment `env:"APP_ENV, required" mapstructure:"environment"`
}

func (b *Base) GetAppEnv() fmt.Stringer {
	return b.Environment
}
