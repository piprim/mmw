package domain

// Parse{{.Name | pascal}}ID parses a string into a {{.Name | pascal}}ID.
func Parse{{.Name | pascal}}ID(id string) ({{.Name | pascal}}ID, error) {
	if id == "" {
		return "", Err{{.Name | pascal}}NotFound
	}
	if len(id) > 255 {
		return "", Err{{.Name | pascal}}NotFound
	}
	return {{.Name | pascal}}ID(id), nil
}

func (id {{.Name | pascal}}ID) String() string { return string(id) }
