package oglslog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/rotisserie/eris"
)

// erisPostPrintHandler wraps a slog.Handler to print stack traces AFTER the log line.
type erisPostPrintHandler struct {
	slog.Handler
}

//nolint:gocritic // because it implements slog.Handle interface
func (h *erisPostPrintHandler) Handle(ctx context.Context, r slog.Record) error {
	// 1. Call the underlying handler FIRST. This prints your standard log line.
	err := h.Handler.Handle(ctx, r)

	if err != nil {
		return fmt.Errorf("eris post Print handler fails: %w", err)
	}

	// 2. Scan the record's attributes for any errors.
	r.Attrs(func(a slog.Attr) bool {
		if e, isError := a.Value.Any().(error); isError {
			// Print the detailed stack trace beautifully below the log line
			red := color.New(color.FgRed)
			red.Printf("↳ Stack Trace for [%s]:\n%s\n", a.Key, eris.ToString(e, true))
		}

		return true // Return true to keep iterating in case there are multiple errors
	})

	return nil
}

// WithAttrs ensures our wrapper survives when you call logger.With()
func (h *erisPostPrintHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &erisPostPrintHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup ensures our wrapper survives when you call logger.WithGroup()
func (h *erisPostPrintHandler) WithGroup(name string) slog.Handler {
	return &erisPostPrintHandler{Handler: h.Handler.WithGroup(name)}
}

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

func New(appEnv string, logLevel slog.Level) (*slog.Logger, error) {
	if appEnv == "" {
		return nil, eris.New("appEnv not set")
	}

	isProd := appEnv == "production"

	replaceErr := func(_ []string, a slog.Attr) slog.Attr {
		if err, isError := a.Value.Any().(error); isError {
			if isProd {
				// Production: Output full structured JSON
				return slog.Any(a.Key, eris.ToJSON(err, true))
			}
			// Local: Keep the log line clean with just the standard error message.
			// The wrapper will handle printing the stack trace below!
			return slog.String(a.Key, err.Error())
		}

		return a
	}

	var llogger *slog.Logger
	if isProd {
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       logLevel,
			ReplaceAttr: replaceErr,
		})
		llogger = slog.New(handler)
	} else {
		// 1. Create your standard local text handler
		baseHandler := StdoutTxtHandler(logLevel, replaceErr)

		// 2. Wrap it with our post-print handler
		llogger = slog.New(&erisPostPrintHandler{Handler: baseHandler})
	}

	return llogger, nil
}
