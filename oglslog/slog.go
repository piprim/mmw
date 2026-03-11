package oglslog

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

// StdoutTxtHandler returns a structured logger to stdout
func StdoutTxtHandler(level slog.Level, replaceAttr func([]string, slog.Attr) slog.Attr) slog.Handler {
	w := os.Stdout
	handler := tint.NewHandler(w, &tint.Options{
		Level:       level,
		TimeFormat:  time.RFC3339Nano,
		NoColor:     !isatty.IsTerminal(w.Fd()),
		AddSource:   true,
		ReplaceAttr: replaceAttr,
	})

	return handler
}

// StdoutLogger returns a structured logger to stderr
func StderrTxtHandler(level slog.Level, replaceAttr func([]string, slog.Attr) slog.Attr) slog.Handler {
	w := os.Stderr
	handler := tint.NewHandler(w, &tint.Options{
		Level:       level,
		TimeFormat:  time.RFC3339Nano,
		NoColor:     !isatty.IsTerminal(w.Fd()),
		AddSource:   true,
		ReplaceAttr: replaceAttr,
	})

	return handler
}

// IOLogger returns a structured logger to `w`
func IOTxtHandler(
	w io.Writer,
	level slog.Level,
	replaceAttr func([]string, slog.Attr) slog.Attr,
	colorized bool,
) slog.Handler {
	handler := tint.NewHandler(w, &tint.Options{
		Level:       level,
		TimeFormat:  time.RFC3339Nano,
		NoColor:     !colorized,
		AddSource:   true,
		ReplaceAttr: replaceAttr,
	})

	return handler
}
