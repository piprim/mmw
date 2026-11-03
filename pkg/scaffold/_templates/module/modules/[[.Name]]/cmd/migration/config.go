package main

import (
	"context"

	"[[.ModulePath]]/internal/infra/config"
)

func loadConfig() (*config.Config, error) {
	//nolint:wrapcheck // Generic loader
	return config.Load(context.Background(), "")
}
