package check

import (
	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/spf13/cobra"
)

func NewFormatCmd() *cobra.Command {
	return newCheckerCmd(
		"format [files...]",
		"Check Go source formatting using gofumpt",
		`Reports any .go file whose content differs from what gofumpt would produce.
No subprocess needed — gofumpt is used as a library.

Machine-generated files are skipped: any file carrying the conventional
"// Code generated ... DO NOT EDIT." comment before its package clause.

Defaults to all tracked *.go files when no arguments are given.`,
		checks.NewFormatChecker,
		"rewrite .go files in-place with gofumpt output",
	)
}
