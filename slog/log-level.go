package oglslog

import "log/slog"

type LogLevel string

// SlogLevel returns the slog.SlogLevel value corresponding to the string level
func (l LogLevel) SlogLevel() slog.Level {
	switch string(l) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo // default to info
	}
}

// String implements the Stringer interface
func (l LogLevel) String() string {
	return string(l)
}

// IsValid checks if the LogLevel value is valid
func (l LogLevel) IsValid() bool {
	switch string(l) {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}
