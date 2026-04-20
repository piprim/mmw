package domain

import (
	"time"

	"github.com/google/uuid"
)

// {{.Name | pascal}} is the aggregate root for the {{.Name}} domain.
type {{.Name | pascal}} struct {
	id        {{.Name | pascal}}ID
	createdAt time.Time
	updatedAt time.Time
	events    []DomainEvent
	userID    uuid.UUID
}

// {{.Name | pascal}}ID is the unique identifier.
type {{.Name | pascal}}ID string

// {{.Name | pascal}}Snapshot is the plain-data representation for persistence.
type {{.Name | pascal}}Snapshot struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID
}

// New{{.Name | pascal}} creates a new {{.Name | pascal}} aggregate.
func New{{.Name | pascal}}(userID uuid.UUID) *{{.Name | pascal}} {
	now := time.Now()
	e := &{{.Name | pascal}}{
		id:        {{.Name | pascal}}ID(uuid.New().String()),
		createdAt: now,
		updatedAt: now,
		events:    []DomainEvent{},
		userID:    userID,
	}
	e.events = append(e.events, {{.Name | pascal}}CreatedEvent{ID: string(e.id)})
	return e
}

func (e *{{.Name | pascal}}) ID() {{.Name | pascal}}ID    { return e.id }
func (e *{{.Name | pascal}}) UserID() uuid.UUID       { return e.userID }
func (e *{{.Name | pascal}}) CreatedAt() time.Time    { return e.createdAt }
func (e *{{.Name | pascal}}) UpdatedAt() time.Time    { return e.updatedAt }

func (e *{{.Name | pascal}}) Snapshot() {{.Name | pascal}}Snapshot {
	return {{.Name | pascal}}Snapshot{
		ID:        uuid.MustParse(string(e.id)),
		CreatedAt: e.createdAt,
		UpdatedAt: e.updatedAt,
		UserID:    e.userID,
	}
}

func Reconstitute{{.Name | pascal}}(snap *{{.Name | pascal}}Snapshot) *{{.Name | pascal}} {
	return &{{.Name | pascal}}{
		id:        {{.Name | pascal}}ID(snap.ID.String()),
		createdAt: snap.CreatedAt,
		updatedAt: snap.UpdatedAt,
		events:    []DomainEvent{},
		userID:    snap.UserID,
	}
}

func (e *{{.Name | pascal}}) PopEvents() []DomainEvent {
	evts := e.events
	e.events = []DomainEvent{}
	return evts
}
