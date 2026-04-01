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

func TestNewErrorLoggingInterceptor_NoError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	interceptor := pfconnect.NewErrorLoggingInterceptor(logger)

	next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&struct{}{}), nil
	})

	req := connect.NewRequest(&struct{}{})
	_, err := interceptor(next)(context.Background(), req)

	require.NoError(t, err)
	assert.Empty(t, buf.String(), "no log should be emitted on success")
}

func TestNewErrorLoggingInterceptor_WithPlainError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	interceptor := pfconnect.NewErrorLoggingInterceptor(logger)

	next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, errors.New("something broke")
	})

	req := connect.NewRequest(&struct{}{})
	_, err := interceptor(next)(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, buf.String(), "handler error")
}

func TestNewErrorLoggingInterceptor_WithConnectError_LogsCause(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	interceptor := pfconnect.NewErrorLoggingInterceptor(logger)

	cause := errors.New("underlying domain error")
	connectErr := connect.NewError(connect.CodeInternal, cause)

	next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, connectErr
	})

	req := connect.NewRequest(&struct{}{})
	_, err := interceptor(next)(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, buf.String(), "handler error")
}

func TestNewErrorLoggingInterceptor_PassesThroughResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	interceptor := pfconnect.NewErrorLoggingInterceptor(logger)

	type payload struct{ Value string }
	expected := &payload{Value: "result"}

	next := connect.UnaryFunc(func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(expected), nil
	})

	req := connect.NewRequest(&struct{}{})
	resp, err := interceptor(next)(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}
