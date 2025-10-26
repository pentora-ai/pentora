// pkg/logging/logging.go
package logging

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// NewLogger creates a new structured logger with component field.
// This is the recommended way to create loggers for dependency injection.
//
// Example usage:
//
//	logger := logging.NewLogger("plugin-service", zerolog.InfoLevel)
//	logger.Info().Msg("Service started")
func NewLogger(component string, level zerolog.Level) zerolog.Logger {
	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("component", component).
		Logger().
		Level(level)
}

// NewLoggerWithWriter creates a logger with a custom writer.
// Useful for testing or custom output destinations.
//
// Example usage:
//
//	logger := logging.NewLoggerWithWriter("test", zerolog.InfoLevel, &bytes.Buffer{})
func NewLoggerWithWriter(component string, level zerolog.Level, w io.Writer) zerolog.Logger {
	return zerolog.New(w).
		With().
		Timestamp().
		Str("component", component).
		Logger().
		Level(level)
}

// ConfigureGlobal sets up the global zerolog logger for CLI usage.
// This should only be called once during CLI initialization.
// For server and service components, prefer using NewLogger with dependency injection.
func ConfigureGlobal(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)

	// Configure console output with timestamp
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05",
	}

	// Set global logger
	log.Logger = zerolog.New(consoleWriter).
		With().
		Timestamp().
		Logger().
		Level(level)
}
