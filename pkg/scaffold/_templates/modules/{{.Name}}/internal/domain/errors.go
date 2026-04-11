package domain

import "errors"

var (
	Err{{.NameTitle}}NotFound    = errors.New("{{.Name}} not found")
	Err{{.NameTitle}}Unavailable = errors.New("{{.Name}} service unavailable")
)
