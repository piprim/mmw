package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

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
func New(db *sql.DB, fsys fs.FS, dir, schemaName string, optionsF ...goose.OptionsFunc) (*Migrator, error) {
	// Set goose to use the provided file system
	goose.SetBaseFS(fsys)
	tableName := schemaName + ".goose_db_version"

	// The schema must exist before goose can create its version-tracking table
	// inside it. Migrations themselves also create the schema (CREATE SCHEMA IF
	// NOT EXISTS), so this is intentionally idempotent.
	if _, err := db.ExecContext(context.Background(),
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName),
	); err != nil {
		return nil, fmt.Errorf("can not create schema %s: %w", schemaName, err)
	}

	return &Migrator{
		db:        db,
		fs:        fsys,
		dir:       dir,
		tableName: tableName,
		optionsF:  optionsF,
	}, nil
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
	normalizedDesc, err := normalizeFileName(description)
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

// unaccentString removes accents from a string using Unicode NFD normalization.
func unaccentString(s string) (string, error) {
	isMn := runes.Predicate(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	})
	transformer := transform.Chain(norm.NFD, runes.Remove(isMn), norm.NFC)
	result, _, err := transform.String(transformer, s)
	if err != nil {
		return "", fmt.Errorf("failed to unaccent string: %w", err)
	}

	return result, nil
}

// normalizeFileName returns a normalized and sanitized filename.
func normalizeFileName(name string) (string, error) {
	if name == "" {
		return "", errors.New("empty name")
	}

	nameUnaccent, err := unaccentString(name)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(nameUnaccent)
	nameSExt := strings.TrimSuffix(nameUnaccent, ext)
	if nameSExt != "" {
		nameSExt = strings.Trim(nameSExt, " ")
		nameSExt = filepath.Clean(strings.ReplaceAll(nameSExt, "..", ""))
		nameSExt = strings.TrimLeft(nameSExt, "/")
		nameSExt = strings.TrimRight(nameSExt, "/")
		nameR := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
		nameSExt = nameR.ReplaceAllString(nameSExt, "-")
		nameR = regexp.MustCompile(`-{2,}`)
		nameSExt = nameR.ReplaceAllString(nameSExt, "-")
	}

	return nameSExt + ext, nil
}
