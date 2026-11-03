package scaffold

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/piprim/goplt"
)

//go:embed all:_templates
var templatesFS embed.FS

func sub(dir string) fs.FS {
	s, err := fs.Sub(templatesFS, "_templates/"+dir)
	if err != nil {
		panic(fmt.Sprintf("scaffold: embedded %s templates FS error: %v", dir, err))
	}

	return s
}

// ModuleFS returns the module-level templates (rendered once per module).
func ModuleFS() fs.FS { return sub("module") }

// SliceFS returns the slice templates (rendered once per entity).
func SliceFS() fs.FS { return sub("slice") }

// MergeManifests returns the module variables followed by the slice
// variables minus Module (known to the module already), with the module's
// delimiters. Conditions are not merged: each FS keeps its own.
func MergeManifests(module, slice *goplt.Manifest) *goplt.Manifest {
	merged := *module
	merged.Variables = append([]goplt.Variable(nil), module.Variables...)
	for _, v := range slice.Variables {
		if v.Name != "Module" {
			merged.Variables = append(merged.Variables, v)
		}
	}

	return &merged
}
