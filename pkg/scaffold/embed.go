package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed all:_templates
var templatesFS embed.FS

// EmbeddedFS returns the embedded templates as an fs.FS with the "_templates/" prefix stripped.
func EmbeddedFS() fs.FS {
	sub, err := fs.Sub(templatesFS, "_templates")
	if err != nil {
		panic(fmt.Sprintf("scaffold: embedded templates FS error: %v", err))
	}

	return sub
}
