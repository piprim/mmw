package domain

import (
	"time"

	"github.com/google/uuid"
)

// Event is the interface that all domain events must implement
type Event interface {
	// EventType returns the type of the event
	EventType() string
	// GetAggregateID returns the ID of the aggregate that emitted the event
	GetAggregateID() string
	// GetUserID returns the ID of the user associated with this event
	GetUserID() uuid.UUID
	// GetOccurredAt returns when the event occurred
	GetOccurredAt() time.Time
}

// BaseEvent contains common fields for all domain events
type BaseEvent struct {
	AggregateID string
	UserID      uuid.UUID
	OccurredAt  time.Time
}

// GetAggregateID returns the ID of the aggregate
func (e BaseEvent) GetAggregateID() string {
	return e.AggregateID
}

// GetUserID returns the ID of the user
func (e BaseEvent) GetUserID() uuid.UUID {
	return e.UserID
}

// GetOccurredAt returns when the event occurred
func (e BaseEvent) GetOccurredAt() time.Time {
	return e.OccurredAt
}
