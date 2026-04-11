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

type {{.NameTitle}}CreatedEvent struct {
	ID string `json:"id"`
}

func (e {{.NameTitle}}CreatedEvent) EventType() string         { return EventTypeCreated }
func (e {{.NameTitle}}CreatedEvent) GetOccurredAt() interface{} { return nil }
