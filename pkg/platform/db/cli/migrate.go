package cli

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/lib/pq" // postgres driver registration required for goose sql-style migrations
	"github.com/piprim/mmw/pkg/platform/db/migrator"
	"github.com/pressly/goose/v3"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

// NewMigrateCmd creates the migrate command with subcommands.
func NewMigrateCmd(m *migrator.Migrator) *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
		},
	}

	ctx := context.Background()

	migrateCmd.AddCommand(migrateUpCmd(ctx, m))
	migrateCmd.AddCommand(migrateDownCmd(ctx, m))
	migrateCmd.AddCommand(migrateVersionCmd(ctx, m))
	migrateCmd.AddCommand(migrateCreateCmd(ctx, m))
	migrateCmd.AddCommand(migrateStatusCmd(ctx, m))
	migrateCmd.AddCommand(migrateFixCmd(ctx, m))
	migrateCmd.AddCommand(migrateResetCmd(ctx, m))

	return migrateCmd
}

func migrateUpCmd(ctx context.Context, m *migrator.Migrator) *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := m.Up(ctx)
			if err != nil {
				return eris.Wrap(err, "error running pending migrations")
			}

			return nil
		},
	}
}

func migrateDownCmd(ctx context.Context, m *migrator.Migrator) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := m.Down(ctx)
			if err != nil {
				return eris.Wrap(err, "error rollbacking the last migration")
			}

			return nil
		},
	}
}

func migrateVersionCmd(ctx context.Context, m *migrator.Migrator) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Retrieves the current migrations' version",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := m.Version(ctx)
			if err != nil {
				return eris.Wrap(err, "error retrieving migrations' version")
			}

			return nil
		},
	}
}

func migrateCreateCmd(ctx context.Context, m *migrator.Migrator) *cobra.Command {
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
			_, _ = fmt.Print("Migration description: ")
			description, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read description: %w", err)
			}
			description = strings.TrimSpace(description)

			// Validate description
			if err := migrator.ValidateDescription(description); err != nil {
				return fmt.Errorf("error retrieving description: %w", err)
			}

			// Prompt for migration type
			_, _ = fmt.Println("Migration type: [1] SQL  [2] Go")
			_, _ = fmt.Print("> ")
			choiceStr, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read migration type: %w", err)
			}
			choiceStr = strings.TrimSpace(choiceStr)

			var mType migrator.MigrationType
			switch choiceStr {
			case "1":
				mType = migrator.SQLMigration
			case "2":
				mType = migrator.GoMigration
			default:
				return eris.New("invalid migration type, choose 1 or 2")
			}

			filePath, err := m.Create(targetDir, description, mType)
			if err != nil {
				return eris.Wrap(err, "error retrieving migrations' version")
			}

			msg := color.GreenString("Migration created:")
			_, _ = fmt.Printf("\n%s %s\n", msg, filePath)

			return nil
		},
	}

	_ = ctx // ctx reserved for future use

	createCmd.Flags().StringVarP(&targetDir, "directory", "d", "", "directory path where live the migrations")

	return createCmd
}

func migrateStatusCmd(ctx context.Context, m *migrator.Migrator) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(_ *cobra.Command, _ []string) error {
			return m.Status(ctx)
		},
	}
}

func migrateFixCmd(ctx context.Context, m *migrator.Migrator) *cobra.Command {
	return &cobra.Command{
		Use:   "fix",
		Short: "Fix migrations' order",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := m.Fix(ctx)
			if err != nil {
				return eris.Wrap(err, "fixing migrations failed")
			}

			return nil
		},
	}
}

func migrateResetCmd(ctx context.Context, m *migrator.Migrator) *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Roll back all migrations",
		RunE: func(_ *cobra.Command, _ []string) error {
			reader := bufio.NewReader(os.Stdin)
			red := color.New(color.FgRed)
			_, _ = red.Println("YOU ARE ABOUT TO ROOL BACK ALL MIGRATIONS!")
			_, _ = red.Print("Are you sure? (yes/No): ")
			choiceStr, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			choiceStr = strings.ToLower(strings.TrimSpace(choiceStr))
			if choiceStr != "yes" {
				_, _ = fmt.Println("\nWise decision. See you later…")

				return nil
			}

			_, err = m.Reset(ctx)
			if err != nil {
				return eris.Wrap(err, "rolling back all migrations")
			}

			return nil
		},
	}
}

// Migrate create and execute the `NewMigrateCmd` command.
// schemaName is the PostgreSQL schema that owns this service's migrations
// (e.g. "auth", "todo"). The goose version table is created inside that schema
// as "<schemaName>.goose_db_version", preventing version collisions when
// multiple services share the same database.
func Migrate(dbURL, schemaName string, migrationsFS fs.FS) error {
	goose.SetLogger(&migrator.FancyLogger{})
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

	options := []goose.OptionsFunc{
		goose.WithAllowMissing(),
	}

	m, err := migrator.New(db, migrationsFS, "scripts", schemaName, options...)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := NewMigrateCmd(m).Execute(); err != nil {
		return fmt.Errorf("failed to exceute migrate command: %w", err)
	}

	return nil
}
