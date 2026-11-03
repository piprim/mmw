package migrations

import "embed"

// Schema is the module's Postgres schema, where goose keeps its version table.
const Schema = "[[.Name]]"

// FS contains all embedded migration files (SQL and Go).
//
//go:embed scripts/*.sql
var FS embed.FS
