package domain

import (
	"time"

	"github.com/google/uuid"
)

// {{.NameTitle}} is the aggregate root for the {{.Name}} domain.
type {{.NameTitle}} struct {
	id        {{.NameTitle}}ID
	createdAt time.Time
	updatedAt time.Time
	events    []DomainEvent
	userID    uuid.UUID
}

// {{.NameTitle}}ID is the unique identifier.
type {{.NameTitle}}ID string

// {{.NameTitle}}Snapshot is the plain-data representation for persistence.
type {{.NameTitle}}Snapshot struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uuid.UUID
}

// New{{.NameTitle}} creates a new {{.NameTitle}} aggregate.
func New{{.NameTitle}}(userID uuid.UUID) *{{.NameTitle}} {
	now := time.Now()
	e := &{{.NameTitle}}{
		id:        {{.NameTitle}}ID(uuid.New().String()),
		createdAt: now,
		updatedAt: now,
		events:    []DomainEvent{},
		userID:    userID,
	}
	e.events = append(e.events, {{.NameTitle}}CreatedEvent{ID: string(e.id)})
	return e
}

func (e *{{.NameTitle}}) ID() {{.NameTitle}}ID    { return e.id }
func (e *{{.NameTitle}}) UserID() uuid.UUID       { return e.userID }
func (e *{{.NameTitle}}) CreatedAt() time.Time    { return e.createdAt }
func (e *{{.NameTitle}}) UpdatedAt() time.Time    { return e.updatedAt }

func (e *{{.NameTitle}}) Snapshot() {{.NameTitle}}Snapshot {
	return {{.NameTitle}}Snapshot{
		ID:        uuid.MustParse(string(e.id)),
		CreatedAt: e.createdAt,
		UpdatedAt: e.updatedAt,
		UserID:    e.userID,
	}
}

func Reconstitute{{.NameTitle}}(snap *{{.NameTitle}}Snapshot) *{{.NameTitle}} {
	return &{{.NameTitle}}{
		id:        {{.NameTitle}}ID(snap.ID.String()),
		createdAt: snap.CreatedAt,
		updatedAt: snap.UpdatedAt,
		events:    []DomainEvent{},
		userID:    snap.UserID,
	}
}

func (e *{{.NameTitle}}) PopEvents() []DomainEvent {
	evts := e.events
	e.events = []DomainEvent{}
	return evts
}
