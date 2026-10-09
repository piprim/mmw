package db

import (
	"github.com/piprim/mmw/internal/pkg/dump"
	"github.com/spf13/cobra"
)

func NewDumpCmd() *cobra.Command {
	var opts dump.Options

	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump a consistent subset of a database as a psql data script",
		Long: `Selects the seed rows of a config file, their children and every row they
reference, and writes them as COPY blocks that psql loads into an empty
database with the same schema. See internal/pkg/dump/README.md.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.Stdout, opts.Stderr = cmd.OutOrStdout(), cmd.ErrOrStderr()

			return dump.Run(cmd.Context(), &opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.URL, "conn", "c", "",
		"connection string, e.g. postgres://user:password@localhost:5432/db?sslmode=disable")
	flags.StringVar(&opts.Config, "config", "", "dump configuration file")
	flags.StringVarP(&opts.Output, "output", "o", "", "output file; stdout when empty")
	_ = cmd.MarkFlagRequired("conn")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}
