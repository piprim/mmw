package slog

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIOTxtHandler(t *testing.T) {
	t.Run("writes message with level and key-value attrs", func(t *testing.T) {
		var buf bytes.Buffer
		handler := IOTxtHandler(&buf, slog.LevelInfo, nil, false)
		logger := slog.New(handler)

		logger.Info("test message", "key", "value")

		output := buf.String()
		assert.Contains(t, output, "INF")
		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "key=value")
	})
}

func TestStdoutTxtHandler(t *testing.T) {
	t.Run("returns non-nil handler", func(t *testing.T) {
		assert.NotNil(t, StdoutTxtHandler(slog.LevelInfo, nil))
	})
}

func TestStderrTxtHandler(t *testing.T) {
	t.Run("returns non-nil handler", func(t *testing.T) {
		assert.NotNil(t, StderrTxtHandler(slog.LevelInfo, nil))
	})
}
