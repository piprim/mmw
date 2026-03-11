package pgcli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/lib/pq"
	"github.com/ovya/ogl/postgres/migrator"
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

	// Setup Migrator
	ctx := context.Background()

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(_ *cobra.Command, _ []string) error {
			version, err := m.Up(ctx)
			if err != nil {
				return eris.Wrap(err, "error running pending migrations")
			}

			fmt.Printf("\nMigrations complete.\nMigrations at version: %d\n", version)

			return nil
		},
	})

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE: func(_ *cobra.Command, _ []string) error {
			version, err := m.Down(ctx)
			if err != nil {
				return eris.Wrap(err, "error rollbacking the last migration")
			}

			fmt.Printf("\nRollback complete. Migrations at version: %d\n", version)

			return nil
		},
	})

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Retrieves the current migrations' version",
		RunE: func(_ *cobra.Command, _ []string) error {
			version, err := m.Version(ctx)
			if err != nil {
				return eris.Wrap(err, "error retrieving migrations' version")
			}

			fmt.Printf("\nMigrations at version: %d\n", version)

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
			if err := migrator.ValidateDescription(description); err != nil {
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

			fmt.Printf("\nMigration created: %s\n", filePath)

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
			version, err := m.Fix(ctx)
			if err != nil {
				return eris.Wrap(err, "fixing migrations failed")
			}

			fmt.Printf("\nMigrations at version: %d\n", version)

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

			version, err := m.Reset(ctx)
			if err != nil {
				return eris.Wrap(err, "rolling back all migrations")
			}

			fmt.Printf("\nMigrations at version: %d\n", version)

			return nil
		},
	})

	return migrateCmd
}
