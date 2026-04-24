package slog

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogLevel_SlogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"unknown", slog.LevelInfo}, // default
		{"", slog.LevelInfo},        // default
		{"DEBUG", slog.LevelInfo},   // case-sensitive, falls to default
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.SlogLevel())
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestLogLevel_IsValid(t *testing.T) {
	tests := []struct {
		level   LogLevel
		isValid bool
	}{
		{"debug", true},
		{"info", true},
		{"warn", true},
		{"error", true},
		{"DEBUG", false},
		{"Info", false},
		{"trace", false},
		{"", false},
		{"verbose", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.level.IsValid())
		})
	}
}
