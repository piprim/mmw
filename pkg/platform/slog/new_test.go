package slog

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates logger with JSON handler", func(t *testing.T) {
		logger, err := New(HandlerJSON, slog.LevelInfo)
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("creates logger with text handler", func(t *testing.T) {
		logger, err := New(HandlerText, slog.LevelDebug)
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})

	t.Run("creates logger at each slog level", func(t *testing.T) {
		for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
			logger, err := New(HandlerJSON, level)
			require.NoError(t, err)
			assert.NotNil(t, logger)
		}
	})
}

func TestErisPostPrintHandler(t *testing.T) {
	t.Run("WithAttrs returns erisPostPrintHandler", func(t *testing.T) {
		var buf bytes.Buffer
		h := &erisPostPrintHandler{Handler: slog.NewTextHandler(&buf, nil)}

		wrapped := h.WithAttrs([]slog.Attr{slog.String("key", "val")})
		require.NotNil(t, wrapped)
		_, ok := wrapped.(*erisPostPrintHandler)
		assert.True(t, ok)
	})

	t.Run("WithGroup returns erisPostPrintHandler", func(t *testing.T) {
		var buf bytes.Buffer
		h := &erisPostPrintHandler{Handler: slog.NewTextHandler(&buf, nil)}

		wrapped := h.WithGroup("mygroup")
		require.NotNil(t, wrapped)
		_, ok := wrapped.(*erisPostPrintHandler)
		assert.True(t, ok)
	})

	t.Run("Handle writes record without error attrs", func(t *testing.T) {
		var buf bytes.Buffer
		h := &erisPostPrintHandler{Handler: slog.NewTextHandler(&buf, nil)}

		r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello world", 0)
		require.NoError(t, h.Handle(context.Background(), r))
		assert.Contains(t, buf.String(), "hello world")
	})

	t.Run("Handle writes record with error attr", func(t *testing.T) {
		var buf bytes.Buffer
		h := &erisPostPrintHandler{Handler: slog.NewTextHandler(&buf, nil)}

		r := slog.NewRecord(time.Now(), slog.LevelError, "something failed", 0)
		r.AddAttrs(slog.Any("err", errors.New("test error")))
		require.NoError(t, h.Handle(context.Background(), r))
		assert.Contains(t, buf.String(), "something failed")
	})

	t.Run("Handle writes record with multiple attrs", func(t *testing.T) {
		var buf bytes.Buffer
		h := &erisPostPrintHandler{Handler: slog.NewTextHandler(&buf, nil)}

		r := slog.NewRecord(time.Now(), slog.LevelError, "multiple errors", 0)
		r.AddAttrs(
			slog.Any("err1", errors.New("first error")),
			slog.String("other", "value"),
		)
		require.NoError(t, h.Handle(context.Background(), r))
		assert.Contains(t, buf.String(), "multiple errors")
	})
}
