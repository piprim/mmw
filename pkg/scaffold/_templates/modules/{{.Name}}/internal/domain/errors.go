package domain

import "errors"

var (
	Err{{.Name | pascal}}NotFound    = errors.New("{{.Name}} not found")
	Err{{.Name | pascal}}Unavailable = errors.New("{{.Name}} service unavailable")
)
