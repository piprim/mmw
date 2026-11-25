package enum

import (
	"errors"
	"fmt"
	"go/token"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config is the content of the enum config file.
type Config struct {
	// Output is the directory the packages are written in,
	// relative to the working directory.
	Output string `toml:"output"`
	// Enums lists the lookup tables to generate an enum for.
	Enums []Entry `toml:"enum"`
}

// LoadConfig reads and validates the TOML file at path.
func LoadConfig(path string) (Config, error) {
	var cfg Config

	file, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("reading config: %w", err)
	}
	defer file.Close()

	if err := toml.NewDecoder(file).DisallowUnknownFields().Decode(&cfg); err != nil {
		var strict *toml.StrictMissingError
		if errors.As(err, &strict) {
			return cfg, fmt.Errorf("%s: unknown key:\n%s", path, strict.String())
		}

		return cfg, fmt.Errorf("%s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}

	return cfg, nil
}

// Validate reports what makes the config unusable: Generate would otherwise write
// nothing, or one entry's file over another's.
func (c Config) Validate() error {
	if c.Output == "" {
		return errors.New("output is required")
	}
	if len(c.Enums) == 0 {
		return errors.New("no [[enum]] entry")
	}

	packages := make(map[string]string, len(c.Enums))
	for i, e := range c.Enums {
		switch {
		case e.Table == "":
			return fmt.Errorf("[[enum]] #%d: table is required", i+1)
		case e.ID == "":
			return fmt.Errorf("[[enum]] %s: id is required", e.Table)
		case e.Label == "":
			return fmt.Errorf("[[enum]] %s: label is required", e.Table)
		}
		// token.IsIdentifier also rejects an empty package and a Go keyword.
		if e.Schema() == "" || e.Name() == "" || !token.IsIdentifier(e.Package()) {
			return fmt.Errorf(
				"[[enum]] %q is not a valid table (want \"table\" or \"schema.table\", giving a Go package name)",
				e.Table)
		}
		if other, ok := packages[e.Package()]; ok {
			return fmt.Errorf("%s and %s give the same package %s", other, e.Table, e.Package())
		}
		packages[e.Package()] = e.Table
	}

	return nil
}
