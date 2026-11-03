// Package testhelpers provides in-memory fakes for the ports of this module.
// Unit, handler and contract tests wire the real application service with
// them, so no mock generation is needed. Each slice adds its own fakes and
// factory in <entity>_fake.go.
package testhelpers

import (
	"context"
	"sync"

	"[[.ModulePath]]/internal/domain"
)
[[- if .Database]]

// --- PassthroughUoW ---

// PassthroughUoW implements ports.UnitOfWork by calling fn(ctx) directly.
// There is no real transaction: suitable only for in-memory tests.
type PassthroughUoW struct{}

func (PassthroughUoW) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// --- Event dispatchers ---

// NoopEventDispatcher implements ports.EventDispatcher by dropping all events.
type NoopEventDispatcher struct{}

func (NoopEventDispatcher) Dispatch(_ context.Context, _ []domain.Event) error {
	return nil
}

// RecordingEventDispatcher implements ports.EventDispatcher by keeping every
// dispatched event, so a test can assert what a command published.
type RecordingEventDispatcher struct {
	mu     sync.Mutex
	Events []domain.Event
}

func (d *RecordingEventDispatcher) Dispatch(_ context.Context, events []domain.Event) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Events = append(d.Events, events...)

	return nil
}
[[- end]]

// --- RecordingEventPublisher ---

// RecordingEventPublisher implements ports.EventPublisher by keeping every
// published event, so a test can assert what a command published.
type RecordingEventPublisher struct {
	mu     sync.Mutex
	Events []domain.Event
}

func (p *RecordingEventPublisher) Publish(_ context.Context, events []domain.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Events = append(p.Events, events...)

	return nil
}
