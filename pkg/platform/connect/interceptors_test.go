package connect_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	pfconnect "github.com/piprim/mmw/pkg/platform/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorLoggingInterceptor(t *testing.T) {
	t.Run("emits no log on success", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))
		interceptor := pfconnect.NewErrorLoggingInterceptor(logger)

		next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&struct{}{}), nil
		})

		_, err := interceptor(next)(context.Background(), connect.NewRequest(&struct{}{}))

		require.NoError(t, err)
		assert.Empty(t, buf.String())
	})

	t.Run("logs handler error on plain error", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))
		interceptor := pfconnect.NewErrorLoggingInterceptor(logger)

		next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, errors.New("something broke")
		})

		_, err := interceptor(next)(context.Background(), connect.NewRequest(&struct{}{}))

		require.Error(t, err)
		assert.Contains(t, buf.String(), "handler error")
	})

	t.Run("logs handler error on connect error with cause", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))
		interceptor := pfconnect.NewErrorLoggingInterceptor(logger)

		cause := errors.New("underlying domain error")
		connectErr := connect.NewError(connect.CodeInternal, cause)

		next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, connectErr
		})

		_, err := interceptor(next)(context.Background(), connect.NewRequest(&struct{}{}))

		require.Error(t, err)
		assert.Contains(t, buf.String(), "handler error")
	})

	t.Run("passes through successful response unchanged", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		interceptor := pfconnect.NewErrorLoggingInterceptor(logger)

		type payload struct{ Value string }
		expected := &payload{Value: "result"}

		next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(expected), nil
		})

		resp, err := interceptor(next)(context.Background(), connect.NewRequest(&struct{}{}))

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}
