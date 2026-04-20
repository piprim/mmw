package domain

const (
	EventTypeCreated = "{{.Name}}.created"
	EventTypeUpdated = "{{.Name}}.updated"
	EventTypeCompleted = "{{.Name}}.completed"
	EventTypeDeleted = "{{.Name}}.deleted"
)

// DomainEvent is the marker interface for all domain events.
type DomainEvent interface {
	EventType() string
	GetOccurredAt() interface{}
}

type {{.Name | pascal}}CreatedEvent struct {
	ID string `json:"id"`
}

func (e {{.Name | pascal}}CreatedEvent) EventType() string         { return EventTypeCreated }
func (e {{.Name | pascal}}CreatedEvent) GetOccurredAt() interface{} { return nil }
