package ieventbus

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPublisher is a mock of message.Publisher
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(topic string, messages ...*message.Message) error {
	args := m.Called(topic, messages)
	return args.Error(0)
}

func (m *MockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewWatermillBus(t *testing.T) {
	mockPub := new(MockPublisher)
	bus := NewWatermillBus(mockPub)
	assert.NotNil(t, bus)
	assert.Equal(t, mockPub, bus.publisher)
}

func TestWatermillBus_Publish(t *testing.T) {
	mockPub := new(MockPublisher)
	bus := NewWatermillBus(mockPub)
	ctx := context.Background()
	payload := []byte("test-payload")
	eventType := "test-event"

	// Expect Publish to be called with the correct topic and a message containing the payload
	mockPub.On("Publish", eventType, mock.MatchedBy(func(msgs []*message.Message) bool {
		if len(msgs) != 1 {
			return false
		}
		msg := msgs[0]
		return string(msg.Payload) == string(payload) && msg.Context() == ctx
	})).Return(nil)

	err := bus.Publish(ctx, eventType, payload)
	assert.NoError(t, err)
	mockPub.AssertExpectations(t)
}

func TestWatermillBus_Publish_Error(t *testing.T) {
	mockPub := new(MockPublisher)
	bus := NewWatermillBus(mockPub)
	ctx := context.Background()
	payload := []byte("test-payload")
	eventType := "test-event"
	expectedErr := errors.New("publish error")

	mockPub.On("Publish", eventType, mock.Anything).Return(expectedErr)

	err := bus.Publish(ctx, eventType, payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "watermillBus publishing error")
	assert.Contains(t, err.Error(), expectedErr.Error())
	mockPub.AssertExpectations(t)
}
