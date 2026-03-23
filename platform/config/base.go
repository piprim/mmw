package oglpfconfig

import (
	"fmt"

	oglconfig "github.com/ovya/ogl/config"
)

type Base struct {
	Environment oglconfig.Environment `env:"APP_ENV, required" mapstructure:"environment"`
}

func (b *Base) GetAppEnv() fmt.Stringer {
	return b.Environment
}
