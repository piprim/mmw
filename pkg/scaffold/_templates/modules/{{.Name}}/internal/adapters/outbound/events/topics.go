package events

import (
	def{{.Name | lower}} "{{.ContractsPath}}/definitions/{{.Name}}"
	"{{.OrgPrefix}}/{{.Name}}/internal/domain"
)

//nolint:gochecknoglobals
var domainTopics = map[string]string{
	domain.EventTypeCreated:   def{{.Name | lower}}.TopicCreated,
	domain.EventTypeUpdated:   def{{.Name | lower}}.TopicUpdated,
	domain.EventTypeCompleted: def{{.Name | lower}}.TopicCompleted,
	domain.EventTypeDeleted:   def{{.Name | lower}}.TopicDeleted,
}
