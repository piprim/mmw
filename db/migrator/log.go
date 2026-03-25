package migrator

import (
	"fmt"
	"log"
	"strings"

	"github.com/fatih/color"
)

// FancyLogger is a colorized logger
// Usage:
// 		goose.SetLogger(&migrator.FancyLogger{})
// 		goose.SetDebug(true)

type FancyLogger struct{}

const grayScale = 200

var (
	ColorOK          = color.New(color.FgGreen)
	ColorComment     = color.New(color.FgCyan)
	ColorGooseOption = color.New(color.FgBlue)
	ColorDefault     = color.RGB(grayScale, grayScale, grayScale)
)

func getColorizer(msg string) *color.Color {
	switch {
	case strings.Contains(msg, "StateMachine: "):
		return nil
	case strings.HasPrefix(msg, "OK ") || strings.Contains(msg, "successfully"):
		return ColorOK
	case strings.HasPrefix(msg, "-- +goose"):
		return ColorGooseOption
	case strings.HasPrefix(msg, "-- "):
		return ColorComment
	default:
		return ColorDefault
	}
}

func (FancyLogger) Printf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	colorizer := getColorizer(msg)
	if colorizer != nil {
		colorizer.Printf(format+"\n", v...)
	}
}

func (FancyLogger) Println(vs ...any) {
	var line string
	for i := range vs {
		if s, ok := vs[i].(string); ok {
			line += s
		} else {
			line += fmt.Sprint(vs[i])
		}
	}

	colorizer := getColorizer(line)
	if colorizer != nil {
		colorizer.Println(line)
	}
}

func (FancyLogger) Fatalf(format string, v ...any) {
	//nolint:revive // FancyLogger implements a logger interface that requires Fatalf
	log.Fatalf(format, v...)
}
