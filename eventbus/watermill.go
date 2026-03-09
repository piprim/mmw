package eventbus

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// WatermillBus implements your workers.SystemEventBus interface
type WatermillBus struct {
	publisher message.Publisher
}

// NewWatermillBus wraps a generic Watermill publisher
func NewWatermillBus(pub message.Publisher) *WatermillBus {
	return &WatermillBus{
		publisher: pub,
	}
}

// Publish converts domain event into a Watermill message and sends it to a topic
func (b *WatermillBus) Publish(ctx context.Context, eventType string, payload []byte) error {
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.SetContext(ctx)
	err := b.publisher.Publish(eventType, msg)
	if err != nil {
		return fmt.Errorf("publishing %s to the topic failed: %w", eventType, err)
	}

	return nil
}
