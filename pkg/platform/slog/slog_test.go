package slog

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIOTxtHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := IOTxtHandler(&buf, slog.LevelInfo, nil, false)
	logger := slog.New(handler)

	logger.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "INF")
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
}

func TestStdoutTxtHandler(t *testing.T) {
	// Cannot easily capture stdout in parallel tests without redirection hacks.
	// We just ensure it doesn't panic and returns a handler.
	handler := StdoutTxtHandler(slog.LevelInfo, nil)
	assert.NotNil(t, handler)
}

func TestStderrTxtHandler(t *testing.T) {
	// Same as above.
	handler := StderrTxtHandler(slog.LevelInfo, nil)
	assert.NotNil(t, handler)
}
