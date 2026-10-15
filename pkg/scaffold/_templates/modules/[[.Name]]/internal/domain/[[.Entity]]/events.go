[[if .Database -]]
package [[.Entity]]

import (
	"time"

	"github.com/google/uuid"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// Event type constants: semantic identifiers owned by the domain. The
// transport topic is mapped once in adapters/outbound/events/topics.go.
const (
	EventTypeDeleted = "[[.Entity]].deleted"
)

// Deleted is emitted when a [[.Entity]] is deleted.
type Deleted struct {
	domain.BaseEvent
	Label string
}

// EventType returns the event type.
func (*Deleted) EventType() string {
	return EventTypeDeleted
}

// newDeletedEvent builds the Deleted event for this aggregate.
func (a *[[.Entity | pascal]]) newDeletedEvent(userID uuid.UUID) *Deleted {
	return &Deleted{
		BaseEvent: domain.BaseEvent{
			AggregateID: a.id.String(),
			UserID:      userID,
			OccurredAt:  time.Now(),
		},
		Label: a.label.String(),
	}
}
[[- else -]]
package [[.Entity]]

import (
	"time"

	"github.com/google/uuid"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// Event type constants: semantic identifiers owned by the domain. The
// transport topic is mapped once in adapters/outbound/events/topics.go.
const (
	EventTypeSent = "[[.Entity]].sent"
)

// Sent is emitted when the gateway accepted a [[.Entity]].
type Sent struct {
	domain.BaseEvent
	Label string
}

// EventType returns the event type.
func (*Sent) EventType() string {
	return EventTypeSent
}

// newSentEvent builds the Sent event for this message.
func (a *[[.Entity | pascal]]) newSentEvent(userID uuid.UUID) *Sent {
	return &Sent{
		BaseEvent: domain.BaseEvent{
			AggregateID: a.label.String(),
			UserID:      userID,
			OccurredAt:  time.Now(),
		},
		Label: a.label.String(),
	}
}
[[- end]]
