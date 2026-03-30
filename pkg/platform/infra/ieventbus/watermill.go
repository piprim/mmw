package ieventbus

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rotisserie/eris"
)

// WatermillBus implements your workers.SystemEventBus interface
type WatermillBus struct {
	publisher message.Publisher
}

// NewWatermillBus wraps any Watermill publisher (GoChannel, Redis, RabbitMQ)
func NewWatermillBus(pub message.Publisher) *WatermillBus {
	return &WatermillBus{
		publisher: pub,
	}
}

// Publish translates your domain's []byte payload into a Watermill message
func (b *WatermillBus) Publish(ctx context.Context, eventType string, payload []byte) error {
	// Create a Watermill message with a unique ID
	msg := message.NewMessage(watermill.NewUUID(), payload)

	// Pass the context along (great for distributed tracing later!)
	msg.SetContext(ctx)

	// Send it to the Watermill engine
	return eris.Wrap(b.publisher.Publish(eventType, msg), "watermillBus publishing error")
}
