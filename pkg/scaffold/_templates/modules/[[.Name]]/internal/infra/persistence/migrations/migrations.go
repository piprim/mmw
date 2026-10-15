package migrations

import "embed"

// FS contains all embedded migration files (SQL and Go).
//
//go:embed scripts/*.sql
var FS embed.FS
