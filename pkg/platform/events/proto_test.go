package events_test

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pfevents "github.com/piprim/mmw/pkg/platform/events"
)

// captureBus captures the last Publish call for inspection.
type captureBus struct {
	topic   string
	payload []byte
}

func (c *captureBus) Publish(_ context.Context, topic string, payload []byte) error {
	c.topic = topic
	c.payload = payload
	return nil
}

func TestPublish_MarshalAndForward(t *testing.T) {
	bus := &captureBus{}
	event := &wrapperspb.StringValue{Value: "hello"}

	err := pfevents.Publish(context.Background(), bus, "test.topic", event)
	require.NoError(t, err)

	assert.Equal(t, "test.topic", bus.topic)

	var decoded wrapperspb.StringValue
	require.NoError(t, protojson.Unmarshal(bus.payload, &decoded))
	assert.Equal(t, "hello", decoded.Value)
}

func TestHandle_UnmarshalAndDelegate(t *testing.T) {
	var received *wrapperspb.StringValue

	handler := pfevents.Handle(func(ctx context.Context, e *wrapperspb.StringValue) error {
		received = e
		return nil
	})

	payload, err := protojson.Marshal(&wrapperspb.StringValue{Value: "world"})
	require.NoError(t, err)

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.SetContext(context.Background())

	err = handler(msg)
	require.NoError(t, err)
	require.NotNil(t, received)
	assert.Equal(t, "world", received.Value)
}

func TestHandle_UnmarshalError(t *testing.T) {
	handler := pfevents.Handle(func(_ context.Context, _ *wrapperspb.StringValue) error {
		return nil
	})

	msg := message.NewMessage(watermill.NewUUID(), []byte("not-valid-json"))
	msg.SetContext(context.Background())

	err := handler(msg)
	assert.ErrorContains(t, err, "unmarshal")
}
