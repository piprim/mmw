package platform_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/piprim/mmw/pkg/platform"
	"github.com/piprim/mmw/pkg/platform/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockModule struct {
	startFn func(ctx context.Context) error
}

func (m *mockModule) Start(ctx context.Context) error {
	return m.startFn(ctx)
}

var _ core.Module = (*mockModule)(nil)

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	app := platform.New(logger, nil)
	assert.NotNil(t, app)
}

func TestApp_Run_NoModules(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	app := platform.New(logger, nil)

	err := app.Run(context.Background())
	assert.NoError(t, err)
}

func TestApp_Run_SuccessfulModule(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	called := false
	mod := &mockModule{startFn: func(_ context.Context) error {
		called = true
		return nil
	}}

	app := platform.New(logger, []core.Module{mod})
	err := app.Run(context.Background())

	require.NoError(t, err)
	assert.True(t, called)
}

func TestApp_Run_ModuleError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	expectedErr := errors.New("module crashed")
	mod := &mockModule{startFn: func(_ context.Context) error {
		return expectedErr
	}}

	app := platform.New(logger, []core.Module{mod})
	err := app.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), expectedErr.Error())
	assert.Contains(t, buf.String(), "application stopped with error")
}

func TestApp_Run_MultipleModules_AllSucceed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	count := 0
	var mu bytes.Buffer // used only for non-data-race sync below
	_ = mu

	mod1 := &mockModule{startFn: func(_ context.Context) error { count++; return nil }}
	mod2 := &mockModule{startFn: func(_ context.Context) error { count++; return nil }}

	app := platform.New(logger, []core.Module{mod1, mod2})
	err := app.Run(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestApp_Run_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	ctx, cancel := context.WithCancel(context.Background())

	mod := &mockModule{startFn: func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}}

	errCh := make(chan error, 1)
	go func() {
		app := platform.New(logger, []core.Module{mod})
		errCh <- app.Run(ctx)
	}()

	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-context.Background().Done():
		t.Fatal("Run did not return after context cancellation")
	}
}
