package oglrunner

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	oglcore "github.com/ovya/ogl/platform/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConfig
type MockConfig struct {
	mock.Mock
}

type stringer string

func (s stringer) String() string { return string(s) }

func (m *MockConfig) GetAppEnv() fmt.Stringer {
	args := m.Called()
	return args.Get(0).(fmt.Stringer)
}
func (m *MockConfig) GetAppName() string {
	args := m.Called()
	return args.String(0)
}
func (m *MockConfig) GetServerPort() string {
	args := m.Called()
	return args.String(0)
}
func (m *MockConfig) GetServerHost() string {
	args := m.Called()
	return args.String(0)
}
func (m *MockConfig) GetDatabaseURL() string {
	args := m.Called()
	return args.String(0)
}

// MockPinger
type MockPinger struct {
	mock.Mock
}

func (m *MockPinger) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockModule
type MockModule struct {
	mock.Mock
}

func (m *MockModule) RegisterRoutes(mux *http.ServeMux) {
	m.Called(mux)
}
func (m *MockModule) Start(ctx context.Context) error {
	args := m.Called(ctx)
	// Block until context is done to simulate worker
	<-ctx.Done()
	return args.Error(0)
}

func (m *MockModule) GetName() string {
	return "mockedApp"
}

func (m *MockModule) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	app := New(logger, nil)
	assert.NotNil(t, app)
	assert.Equal(t, logger, app.logger)
}

func TestApp_Run(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mod := new(MockModule)
	mod.On("Start", mock.Anything).Return(nil)

	app := New(logger, []oglcore.Module{mod})

	// Run
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)

	go func() {
		errChan <- app.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop
	cancel()

	// Wait for return
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("App.Run did not return after context cancellation")
	}

	mod.AssertExpectations(t)
}
