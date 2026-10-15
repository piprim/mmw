package main

import (
	"context"

	"[[.OrgPrefix]]/[[.Name]]/internal/infra/config"
	"[[.OrgPrefix]]/[[.Name]]/internal/infra/persistence/migrations"
)

var migrationsFS = migrations.FS

func loadConfig() (*config.Config, error) {
	//nolint:wrapcheck // Generic loader
	return config.Load(context.Background(), "")
}
