package custom

import (
	"fmt"
	"os"
	"path/filepath"
)

// ApplicationPurityValidator checks that internal/application/ never imports contracts/.
type ApplicationPurityValidator struct {
	ModulesDir string
}

func (v *ApplicationPurityValidator) Name() string { return "application-purity" }

func (v *ApplicationPurityValidator) Description() string {
	return "internal/application/ must not import contracts/ (proto concerns belong in adapters)"
}

func (v *ApplicationPurityValidator) Check() error {
	entries, err := os.ReadDir(v.ModulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read modules dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		appDir := filepath.Join(v.ModulesDir, entry.Name(), "internal", "application")
		if _, err := os.Stat(appDir); os.IsNotExist(err) {
			continue
		}
		if err := walkForForbiddenImport(appDir, "contracts", entry.Name(), "application"); err != nil {
			return err
		}
	}
	return nil
}
