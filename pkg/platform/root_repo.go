package platform

import (
	"os"
	"path/filepath"
)

// RootRepo walks up from the working directory to find go.work.
func RootRepo() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "."
}
