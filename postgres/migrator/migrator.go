package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	oglstring "github.com/ovya/ogl/oglstring"
	"github.com/pressly/goose/v3"
)

type MigrationType int

const (
	SQLMigration MigrationType = iota
	GoMigration
)

// Migrator handles database migrations independent of the CLI
type Migrator struct {
	db       *sql.DB
	fs       fs.FS
	dir      string
	optionsF []goose.OptionsFunc
}

// New creates a new Migrator.
// Pass os.DirFS(dir) for local development, or your embedded fs.FS for production.
func New(db *sql.DB, fsys fs.FS, dir string, optionsF ...goose.OptionsFunc) *Migrator {
	// Set goose to use the provided file system
	goose.SetBaseFS(fsys)
	_ = goose.SetDialect("postgres")

	return &Migrator{
		db:       db,
		fs:       fsys,
		dir:      dir,
		optionsF: optionsF,
	}
}

func (m *Migrator) Up(ctx context.Context) (int64, error) {
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
	if err := goose.StatusContext(ctx, m.db, m.dir); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (m *Migrator) Version(ctx context.Context) (int64, error) {
	i, err := goose.GetDBVersionContext(ctx, m.db)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	return i, nil
}

// Create generates a new migration file. This strictly writes to the local disk.
func (m *Migrator) Create(targetDir, description string, mType MigrationType) (string, error) {
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
	if err := os.MkdirAll(targetDir, 0755); err != nil {
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
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
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
		return fmt.Errorf("description is required")
	}

	if len(description) < 3 || len(description) > 100 {
		return fmt.Errorf("description must be between 3 and 100 characters")
	}

	return nil
}

// toCamelCase converts a string to CamelCase for Go function names.
// Handles dashes, underscores, and spaces as separators.
func toCamelCase(s string) string {
	// Replace dashes and underscores with spaces for uniform processing
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// Split by spaces
	words := strings.Fields(s)

	// Capitalize first letter of each word
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, "")
}
