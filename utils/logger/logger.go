package logger

import (
	"os"

	"github.com/0xA2618/logjson"
)

// Log is the global logger instance
var Log *logjson.Logger

// Init initializes the global logger with application context
// mode: "dev" for debug level, otherwise info level
func Init(mode string) {
	// Determine log level from mode
	level := logjson.LevelInfo
	if mode == "dev" {
		level = logjson.LevelDebug
	}

	Log = logjson.New(
		logjson.WithOutput(os.Stdout),
		logjson.WithLevel(level),
		logjson.WithField("app", "IndicatorTask"),
	)
}
