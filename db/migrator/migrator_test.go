package oglmigrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     bool
	}{
		{
			name:        "valid description",
			description: "create users table",
			wantErr:     false,
		},
		{
			name:        "empty description",
			description: "",
			wantErr:     true,
		},
		{
			name:        "short description",
			description: "ab",
			wantErr:     true,
		},
		{
			name:        "long description",
			description: string(make([]byte, 101)),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDescription(tt.description)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"create users", "CreateUsers"},
		{"add_column_to_users", "AddColumnToUsers"},
		{"drop-table-posts", "DropTablePosts"},
		{"mixed_separators test", "MixedSeparatorsTest"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMigrator_Create(t *testing.T) {
	// Create temp dir for migrations
	tmpDir, err := os.MkdirTemp("", "migrations-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// We don't need a DB connection for Create
	m := New(nil, nil, tmpDir, "test.goose_db_version")

	t.Run("Create SQL migration", func(t *testing.T) {
		desc := "create items table"
		path, err := m.Create(tmpDir, desc, SQLMigration)
		assert.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.FileExists(t, path)
		assert.Contains(t, filepath.Base(path), "_create-items-table.sql")

		content, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "-- +goose Up")
	})

	t.Run("Create Go migration", func(t *testing.T) {
		desc := "add default user"
		path, err := m.Create(tmpDir, desc, GoMigration)
		assert.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.FileExists(t, path)
		assert.Contains(t, filepath.Base(path), "_add-default-user.go")

		filename := filepath.Base(path)
		timestamp := strings.Split(filename, "_")[0]

		content, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "func up"+timestamp)
		assert.Contains(t, string(content), "func down"+timestamp)
	})

	t.Run("Create with invalid description", func(t *testing.T) {
		_, err := m.Create(tmpDir, "", SQLMigration)
		assert.Error(t, err)
	})
}

func TestRenderTemplates(t *testing.T) {
	t.Run("renderSQLTemplate", func(t *testing.T) {
		out, err := renderSQLTemplate()
		assert.NoError(t, err)
		assert.Contains(t, out, "-- +goose Up")
		assert.Contains(t, out, "-- +goose Down")
	})

	t.Run("renderGoTemplate", func(t *testing.T) {
		out, err := renderGoTemplate("my test migration")
		assert.NoError(t, err)
		assert.Contains(t, out, "package migrations")
		assert.Contains(t, out, "func upMyTestMigration")
		assert.Contains(t, out, "func downMyTestMigration")
		assert.Contains(t, out, "goose.AddMigrationContext(upMyTestMigration, downMyTestMigration)")
	})
}
