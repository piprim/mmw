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

func TestNew_JSONHandler(t *testing.T) {
	logger, err := New(HandlerJSON, slog.LevelInfo)
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestNew_TextHandler(t *testing.T) {
	logger, err := New(HandlerText, slog.LevelDebug)
	require.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestNew_AllLevels(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		logger, err := New(HandlerJSON, level)
		require.NoError(t, err)
		assert.NotNil(t, logger)
	}
}

func TestErisPostPrintHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, nil)
	h := &erisPostPrintHandler{Handler: base}

	wrapped := h.WithAttrs([]slog.Attr{slog.String("key", "val")})
	require.NotNil(t, wrapped)

	_, ok := wrapped.(*erisPostPrintHandler)
	assert.True(t, ok, "WithAttrs should return an *erisPostPrintHandler")
}

func TestErisPostPrintHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, nil)
	h := &erisPostPrintHandler{Handler: base}

	wrapped := h.WithGroup("mygroup")
	require.NotNil(t, wrapped)

	_, ok := wrapped.(*erisPostPrintHandler)
	assert.True(t, ok, "WithGroup should return an *erisPostPrintHandler")
}

func TestErisPostPrintHandler_Handle_NoError(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, nil)
	h := &erisPostPrintHandler{Handler: base}

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello world", 0)
	err := h.Handle(context.Background(), r)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "hello world")
}

func TestErisPostPrintHandler_Handle_WithError(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, nil)
	h := &erisPostPrintHandler{Handler: base}

	r := slog.NewRecord(time.Now(), slog.LevelError, "something failed", 0)
	r.AddAttrs(slog.Any("err", errors.New("test error")))

	err := h.Handle(context.Background(), r)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "something failed")
}

func TestErisPostPrintHandler_Handle_MultipleErrors(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, nil)
	h := &erisPostPrintHandler{Handler: base}

	r := slog.NewRecord(time.Now(), slog.LevelError, "multiple errors", 0)
	r.AddAttrs(
		slog.Any("err1", errors.New("first error")),
		slog.String("other", "value"),
	)

	err := h.Handle(context.Background(), r)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "multiple errors")
}
