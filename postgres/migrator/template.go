package migrator

import (
	"bytes"
	"fmt"
	"text/template"
)

const sqlMigrationTemplate = `-- +goose Up
-- +goose StatementBegin
-- TODO: Add your schema changes here

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- TODO: Add rollback logic here

-- +goose StatementEnd
`

const goMigrationTemplate = `package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

//nolint:gochecknoinits // Goose wants it
func init() {
	goose.AddMigrationContext(up{{.CamelCase}}, down{{.CamelCase}})
}

func up{{.CamelCase}}(ctx context.Context, tx *sql.Tx) error {
	// Implement migration logic here
	return nil
}

func down{{.CamelCase}}(ctx context.Context, tx *sql.Tx) error {
	// Implement rollback logic here
	return nil
}
`

// renderSQLTemplate renders the SQL migration template.
func renderSQLTemplate() (string, error) {
	return sqlMigrationTemplate, nil
}

// renderGoTemplate renders the Go migration template with the given description.
func renderGoTemplate(description string) (string, error) {
	tmpl, err := template.New("migration").Parse(goMigrationTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := map[string]string{
		"CamelCase": toCamelCase(description),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
