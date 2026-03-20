package oglpgcli

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/lib/pq"
	oglmigrator "github.com/ovya/ogl/db/migrator"
	"github.com/pressly/goose/v3"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

// NewMigrateCmd creates the migrate command with subcommands.
func NewMigrateCmd(m *oglmigrator.Migrator) *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
		},
	}

	// Setup Migrator
	ctx := context.Background()

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := m.Up(ctx)
			if err != nil {
				return eris.Wrap(err, "error running pending migrations")
			}

			return nil
		},
	})

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := m.Down(ctx)
			if err != nil {
				return eris.Wrap(err, "error rollbacking the last migration")
			}

			return nil
		},
	})

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Retrieves the current migrations' version",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := m.Version(ctx)
			if err != nil {
				return eris.Wrap(err, "error retrieving migrations' version")
			}

			return nil
		},
	})

	var targetDir string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new migration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmd.MarkFlagRequired("directory"); err != nil {
				return fmt.Errorf("%w", err)
			}

			reader := bufio.NewReader(os.Stdin)

			// Prompt for description
			fmt.Print("Migration description: ")
			description, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}
			description = strings.TrimSpace(description)

			// Validate description
			if err := oglmigrator.ValidateDescription(description); err != nil {
				return fmt.Errorf("error retrieving description: %w", err)
			}

			// Prompt for migration type
			fmt.Println("Migration type: [1] SQL  [2] Go")
			fmt.Print("> ")
			choiceStr, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read migration type: %w", err)
			}
			choiceStr = strings.TrimSpace(choiceStr)

			var mType oglmigrator.MigrationType
			switch choiceStr {
			case "1":
				mType = oglmigrator.SQLMigration
			case "2":
				mType = oglmigrator.GoMigration
			default:
				return eris.New("invalid migration type, choose 1 or 2")
			}

			filePath, err := m.Create(targetDir, description, mType)
			if err != nil {
				return eris.Wrap(err, "error retrieving migrations' version")
			}

			msg := color.GreenString("Migration created:")
			fmt.Printf("\n%s %s\n", msg, filePath)

			return nil
		},
	}

	createCmd.Flags().StringVarP(&targetDir, "directory", "d", "", "directory path where live the migrations")

	migrateCmd.AddCommand(createCmd)

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(_ *cobra.Command, _ []string) error {
			return m.Status(ctx)
		},
	})

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "fix",
		Short: "Fix migrations' order",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := m.Fix(ctx)
			if err != nil {
				return eris.Wrap(err, "fixing migrations failed")
			}

			return nil
		},
	})

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Roll back all migrations",
		RunE: func(_ *cobra.Command, _ []string) error {
			reader := bufio.NewReader(os.Stdin)
			red := color.New(color.FgRed)
			red.Println("YOU ARE ABOUT TO ROOL BACK ALL MIGRATIONS!")
			red.Print("Are you sure? (yes/No): ")
			choiceStr, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			choiceStr = strings.ToLower(strings.TrimSpace(choiceStr))
			if choiceStr != "yes" {
				fmt.Println("\nWise decision. See you later…")
				return nil
			}

			_, err = m.Reset(ctx)
			if err != nil {
				return eris.Wrap(err, "rolling back all migrations")
			}

			return nil
		},
	})

	return migrateCmd
}

// Migrate create and execute the `NewMigrateCmd` command.
// schemaName is the PostgreSQL schema that owns this service's migrations
// (e.g. "auth", "todo"). The goose version table is created inside that schema
// as "<schemaName>.goose_db_version", preventing version collisions when
// multiple services share the same database.
func Migrate(dbURL, schemaName string, migrationsFS fs.FS) error {
	goose.SetLogger(&oglmigrator.FancyLogger{})
	goose.SetDebug(true)
	goose.SetSequential(true)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("can not open database connection: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		return fmt.Errorf("can ping database connection: %w", err)
	}

	// The schema must exist before goose can create its version-tracking table
	// inside it. Migrations themselves also create the schema (CREATE SCHEMA IF
	// NOT EXISTS), so this is intentionally idempotent.
	if _, err := db.ExecContext(context.Background(),
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName),
	); err != nil {
		return fmt.Errorf("can not create schema %s: %w", schemaName, err)
	}

	options := []goose.OptionsFunc{
		goose.WithAllowMissing(),
	}

	tableName := schemaName + ".goose_db_version"
	m := oglmigrator.New(db, migrationsFS, "scripts", tableName, options...)

	if err := NewMigrateCmd(m).Execute(); err != nil {
		return fmt.Errorf("failed to exceute migrate command: %w", err)
	}

	return nil
}
