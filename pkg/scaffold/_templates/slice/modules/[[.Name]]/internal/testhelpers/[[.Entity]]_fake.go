[[if ne .Shape "gateway" -]]
package testhelpers

import (
	"context"
	"sync"
	"testing"

	"[[.ModulePath]]/internal/application"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

// --- InMemory[[.Entity | pascal]]Repo ---

// InMemory[[.Entity | pascal]]Repo is a map-backed ports.[[.Entity | pascal]]Repository.
// It returns [[.Entity]].ErrNotFound when an item is absent.
type InMemory[[.Entity | pascal]]Repo struct {
	mu    sync.RWMutex
	items [[printf "map[%s.ID]" .Entity]]*[[.Entity]].[[.Entity | pascal]]
}

func NewInMemory[[.Entity | pascal]]Repo() *InMemory[[.Entity | pascal]]Repo {
	return &InMemory[[.Entity | pascal]]Repo{items: make([[printf "map[%s.ID]" .Entity]]*[[.Entity]].[[.Entity | pascal]])}
}

// Seed stores a [[.Entity]] so tests can pre-populate state (there is no Save port yet).
func (r *InMemory[[.Entity | pascal]]Repo) Seed(a *[[.Entity]].[[.Entity | pascal]]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[a.[[if eq .Shape "read-model"]]ID[[else]]ID()[[end]]] = a
}

// Len returns the number of stored rows.
func (r *InMemory[[.Entity | pascal]]Repo) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.items)
}

func (r *InMemory[[.Entity | pascal]]Repo) [[.Entity | pascal]]FindByID(
	_ context.Context, id [[.Entity]].ID,
) (*[[.Entity]].[[.Entity | pascal]], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.items[id]
	if !ok {
		return nil, [[.Entity]].ErrNotFound
	}

	return a, nil
}

[[- if eq .Shape "aggregate"]]

func (r *InMemory[[.Entity | pascal]]Repo) [[.Entity | pascal]]Delete(_ context.Context, id [[.Entity]].ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return [[.Entity]].ErrNotFound
	}
	delete(r.items, id)

	return nil
}
[[- end]]

func (r *InMemory[[.Entity | pascal]]Repo) Health(_ context.Context) (any, error) {
	return r.Len(), nil
}

[[- if eq .Shape "read-model"]]

// NewTest[[.Entity | pascal]]Service wires a real [[.Entity | pascal]]AppService with an in-memory
// repository. It returns the repo so tests can seed state.
func NewTest[[.Entity | pascal]]Service(
	t *testing.T,
) (application.[[.Entity | pascal]]Service, *InMemory[[.Entity | pascal]]Repo) {
	t.Helper()
	repo := NewInMemory[[.Entity | pascal]]Repo()

	return application.New[[.Entity | pascal]]AppService(repo), repo
}
[[- else]]

// NewTest[[.Entity | pascal]]Service wires a real [[.Entity | pascal]]AppService with in-memory
// fakes. It returns the repo and the dispatcher so tests can seed state and
// assert the published events.
func NewTest[[.Entity | pascal]]Service(
	t *testing.T,
) (application.[[.Entity | pascal]]Service, *InMemory[[.Entity | pascal]]Repo, *RecordingEventDispatcher) {
	t.Helper()
	repo := NewInMemory[[.Entity | pascal]]Repo()
	dispatcher := &RecordingEventDispatcher{}
	svc := application.New[[.Entity | pascal]]AppService(repo, PassthroughUoW{}, dispatcher)

	return svc, repo, dispatcher
}
[[- end]]
[[- else -]]
package testhelpers

import (
	"context"
	"sync"
	"testing"

	"[[.ModulePath]]/internal/application"
	"[[.ModulePath]]/internal/domain/[[.Entity]]"
)

// --- Fake[[.Entity | pascal]]Gateway ---

// Fake[[.Entity | pascal]]Gateway implements ports.[[.Entity | pascal]]Gateway by
// recording every message; set Err to make it fail.
type Fake[[.Entity | pascal]]Gateway struct {
	mu   sync.Mutex
	Sent []*[[.Entity]].[[.Entity | pascal]]
	Err  error
}

func (g *Fake[[.Entity | pascal]]Gateway) [[.Entity | pascal]]Send(_ context.Context, a *[[.Entity]].[[.Entity | pascal]]) error {
	if g.Err != nil {
		return g.Err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Sent = append(g.Sent, a)

	return nil
}

func (g *Fake[[.Entity | pascal]]Gateway) Health(_ context.Context) (any, error) {
	if g.Err != nil {
		return nil, g.Err
	}

	return "ok", nil
}

// NewTest[[.Entity | pascal]]Service wires a real [[.Entity | pascal]]AppService with fakes.
// It returns the gateway and the publisher so tests can assert what was sent
// and published.
func NewTest[[.Entity | pascal]]Service(
	t *testing.T,
) (application.[[.Entity | pascal]]Service, *Fake[[.Entity | pascal]]Gateway, *RecordingEventPublisher) {
	t.Helper()
	gateway := &Fake[[.Entity | pascal]]Gateway{}
	publisher := &RecordingEventPublisher{}
	svc := application.New[[.Entity | pascal]]AppService(gateway, publisher)

	return svc, gateway, publisher
}
[[- end]]
