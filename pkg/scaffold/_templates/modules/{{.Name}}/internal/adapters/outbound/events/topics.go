package events

import (
	{{.PkgDef}} "{{.ContractsPath}}/definitions/{{.Name}}"
	"{{.ModulePath}}/internal/domain"
)

//nolint:gochecknoglobals
var domainTopics = map[string]string{
	domain.EventTypeCreated:   {{.PkgDef}}.TopicCreated,
	domain.EventTypeUpdated:   {{.PkgDef}}.TopicUpdated,
	domain.EventTypeCompleted: {{.PkgDef}}.TopicCompleted,
	domain.EventTypeDeleted:   {{.PkgDef}}.TopicDeleted,
}
