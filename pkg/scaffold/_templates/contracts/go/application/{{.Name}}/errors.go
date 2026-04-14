package {{.PkgDef}}

import "errors"

var (
	Err{{.NameTitle}}NotFound    = errors.New("{{.Name}} not found")
	Err{{.NameTitle}}Unavailable = errors.New("{{.Name}} service unavailable")
)

// Topic constants for Watermill routing keys.
const (
	TopicCreated   = "{{.Name}}.created"
	TopicUpdated   = "{{.Name}}.updated"
	TopicCompleted = "{{.Name}}.completed"
	TopicDeleted   = "{{.Name}}.deleted"
)
