package domain

import "github.com/google/uuid"

// Parse{{.NameTitle}}ID parses a string into a {{.NameTitle}}ID.
func Parse{{.NameTitle}}ID(id string) ({{.NameTitle}}ID, error) {
	if id == "" {
		return "", Err{{.NameTitle}}NotFound
	}
	if _, err := uuid.Parse(id); err != nil {
		return "", Err{{.NameTitle}}NotFound
	}
	return {{.NameTitle}}ID(id), nil
}

func (id {{.NameTitle}}ID) String() string { return string(id) }
