package events

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Publish marshals event as protojson and publishes it on topic.
func Publish[T proto.Message](ctx context.Context, bus SystemEventBus, topic string, event T) error {
	b, err := protojson.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal %T: %w", event, err)
	}

	if err := bus.Publish(ctx, topic, b); err != nil {
		return fmt.Errorf("failed to publish event %T: %w", event, err)
	}

	return nil
}

// Handle wraps a typed proto handler into a Watermill function that returns only an error.
// Useful for router.AddNoPublisherHandler.
//
// The double type parameter constraint (T = *PT) lets Go allocate a concrete
// zero value with new(PT) and then convert it to the interface T, avoiding
// a reflect-based allocation.
//
// Usage:
//
//	router.AddNoPublisherHandler("name", topic, sub,
//	    pfevents.Handle(func(ctx context.Context, e *defauth.UserDeletedEvent) error {
//	        return cmd.Execute(ctx, e.UserId)
//	    }),
//	)
func Handle[T interface {
	proto.Message
	*PT
}, PT any](fn func(context.Context, T) error) func(*message.Message) error {
	return func(msg *message.Message) error {
		event := T(new(PT))
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(msg.Payload, event); err != nil {
			return fmt.Errorf("unmarshal %T: %w", event, err)
		}

		return fn(msg.Context(), event)
	}
}
