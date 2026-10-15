[[if .Database -]]
package [[.Entity]]

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// ID is the integer primary key of a [[.Entity]] ([[.Name]].[[.Entity]].id).
type ID int64

// ParseID validates a raw integer and returns an ID.
// Postgres sequences start at 1, so zero and negative values can never match a row.
func ParseID(id int64) (ID, error) {
	if id <= 0 {
		return 0, ErrInvalidID
	}

	return ID(id), nil
}

// Int64 returns the primitive representation of ID.
func (id ID) Int64() int64 {
	return int64(id)
}

// String returns the ID as decimal text, the form events carry.
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// [[.Entity | pascal]] is the aggregate root of the [[.Entity]] domain. Rows of
// [[.Name]].[[.Entity]] are created by migrations or by a future create command;
// the only state change so far is deletion, which records a domain event.
//
// There is no New constructor yet: add one (with a Created event) together
// with the create command, following domain/todo in modules/core.
type [[.Entity | pascal]] struct {
	id        ID
	label     Label
	createdAt time.Time
	updatedAt time.Time
	events    []domain.Event
}

// ID returns the [[.Entity]] ID.
func (a *[[.Entity | pascal]]) ID() ID {
	return a.id
}

// Label returns the label.
func (a *[[.Entity | pascal]]) Label() Label {
	return a.label
}

// CreatedAt returns when the [[.Entity]] was created.
func (a *[[.Entity | pascal]]) CreatedAt() time.Time {
	return a.createdAt
}

// UpdatedAt returns when the [[.Entity]] was last updated.
func (a *[[.Entity | pascal]]) UpdatedAt() time.Time {
	return a.updatedAt
}

// Events returns the unpublished domain events.
func (a *[[.Entity | pascal]]) Events() []domain.Event {
	return a.events
}

// ClearEvents clears the unpublished events (called after publishing).
func (a *[[.Entity | pascal]]) ClearEvents() {
	a.events = []domain.Event{}
}

// Delete records the Deleted event on behalf of userID. The caller removes
// the row from the repository and dispatches the events in one transaction.
func (a *[[.Entity | pascal]]) Delete(userID uuid.UUID) {
	a.addEvent(a.newDeletedEvent(userID))
}

// addEvent adds a domain event to the unpublished events list.
func (a *[[.Entity | pascal]]) addEvent(event domain.Event) {
	a.events = append(a.events, event)
}
[[- else -]]
package [[.Entity]]

import (
	"github.com/google/uuid"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain"
)

// [[.Entity | pascal]] is a message this module sends through an external
// gateway. It has no identity of its own: nothing is stored, and the event
// it raises carries the label as aggregate ID.
type [[.Entity | pascal]] struct {
	label  Label
	events []domain.Event
}

// New builds a [[.Entity]] ready to send.
func New(label Label) *[[.Entity | pascal]] {
	return &[[.Entity | pascal]]{label: label}
}

// Label returns the label.
func (a *[[.Entity | pascal]]) Label() Label {
	return a.label
}

// Events returns the unpublished domain events.
func (a *[[.Entity | pascal]]) Events() []domain.Event {
	return a.events
}

// ClearEvents clears the unpublished events (called after publishing).
func (a *[[.Entity | pascal]]) ClearEvents() {
	a.events = []domain.Event{}
}

// MarkSent records the Sent event on behalf of userID, after the gateway
// accepted the message.
func (a *[[.Entity | pascal]]) MarkSent(userID uuid.UUID) {
	a.events = append(a.events, a.newSentEvent(userID))
}
[[- end]]
