package events

import "context"

// SystemEventBus represents the actual transport mechanism across the mmw
// e.g., an in-memory channel broker, NATS, or RabbitMQ.
type SystemEventBus interface {
	// Publish sends an event with the given type and payload to the bus.
	Publish(ctx context.Context, eventType string, payload []byte) error
}
