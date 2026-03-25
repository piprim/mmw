package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	oglstring "github.com/ovya/ogl/string"
	"github.com/pressly/goose/v3"
)

type MigrationType int

const (
	SQLMigration MigrationType = iota
	GoMigration
)

// Migrator handles database migrations independent of the CLI
type Migrator struct {
	db        *sql.DB
	fs        fs.FS
	dir       string
	tableName string
	optionsF  []goose.OptionsFunc
}

// New creates a new Migrator.
// Pass os.DirFS(dir) for local development, or your embedded fs.FS for production.
// tableName is the schema-qualified table used to track migration versions (e.g. "auth.goose_db_version").
// Using a per-service schema-qualified name prevents version collisions when multiple services
// share the same database.
func New(db *sql.DB, fsys fs.FS, dir, tableName string, optionsF ...goose.OptionsFunc) *Migrator {
	// Set goose to use the provided file system
	goose.SetBaseFS(fsys)

	return &Migrator{
		db:        db,
		fs:        fsys,
		dir:       dir,
		tableName: tableName,
		optionsF:  optionsF,
	}
}

func (m *Migrator) Up(ctx context.Context) (int64, error) {
	goose.SetTableName(m.tableName)

	if err := goose.UpContext(ctx, m.db, m.dir, m.optionsF...); err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	i, err := goose.GetDBVersionContext(ctx, m.db)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	return i, nil
}

func (m *Migrator) Down(ctx context.Context) (int64, error) {
	goose.SetTableName(m.tableName)

	if err := goose.DownContext(ctx, m.db, m.dir); err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	i, err := goose.GetDBVersionContext(ctx, m.db)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	return i, nil
}

func (m *Migrator) Fix(ctx context.Context) (int64, error) {
	goose.SetTableName(m.tableName)

	if err := goose.Fix(m.dir); err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	i, err := goose.GetDBVersionContext(ctx, m.db)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	return i, nil
}

func (m *Migrator) Reset(ctx context.Context) (int64, error) {
	goose.SetTableName(m.tableName)

	if err := goose.Reset(m.db, m.dir, m.optionsF...); err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	i, err := goose.GetDBVersionContext(ctx, m.db)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	return i, nil
}

func (m *Migrator) Status(ctx context.Context) error {
	goose.SetTableName(m.tableName)

	if err := goose.StatusContext(ctx, m.db, m.dir); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (m *Migrator) Version(ctx context.Context) (int64, error) {
	goose.SetTableName(m.tableName)

	i, err := goose.GetDBVersionContext(ctx, m.db)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	return i, nil
}

// Create generates a new migration file. This strictly writes to the local disk.
func (*Migrator) Create(targetDir, description string, mType MigrationType) (string, error) {
	// Validate description
	if err := ValidateDescription(description); err != nil {
		return "", fmt.Errorf("migration description is not valid: %w", err)
	}

	// Normalize description for filename using existing NormalizeFileName
	normalizedDesc, err := oglstring.NormalizeFileName(description)
	if err != nil {
		return "", fmt.Errorf("failed to normalize description: %w", err)
	}

	// If suffix = "_test" the file is considered as a test and it is ignored.
	if normalizedDesc == "test" {
		normalizedDesc = "try"
	}

	// Generate timestamp
	timestamp := time.Now().Format("20060102150405")

	// Generate filename
	var filename string
	if mType == SQLMigration {
		filename = fmt.Sprintf("%s_%s.sql", timestamp, normalizedDesc)
	} else {
		filename = fmt.Sprintf("%s_%s.go", timestamp, normalizedDesc)
	}

	filePath := filepath.Join(targetDir, filename)

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Render template
	var content string
	if mType == SQLMigration {
		content, err = renderSQLTemplate()
	} else {
		content, err = renderGoTemplate(timestamp)
	}
	if err != nil {
		return "", err
	}

	//
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("failed to write migration file: %w", err)
	}

	// Get absolute path for output
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	return absPath, nil
}

// ValidateDescription validates a migration description.
// Returns error if description is invalid.
func ValidateDescription(description string) error {
	if description == "" {
		return errors.New("description is required")
	}

	if len(description) < 3 || len(description) > 100 {
		return errors.New("description must be between 3 and 100 characters")
	}

	return nil
}

// toCamelCase converts a string to CamelCase for Go function names.
// Handles dashes, underscores, and spaces as separators.
func toCamelCase(s string) string {
	// Replace dashes and underscores with spaces for uniform processing
	v := strings.ReplaceAll(s, "-", " ")
	v = strings.ReplaceAll(v, "_", " ")

	// Split by spaces
	words := strings.Fields(v)

	// Capitalize first letter of each word
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, "")
}
