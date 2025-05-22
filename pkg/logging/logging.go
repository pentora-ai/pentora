// pkg/logging/logging.go
package logging

import (
	"io"
	stdLog "log"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// logLevel stores the current log level globally
	logLevel zerolog.Level
	// logWriter stores the current log writer globally
	logWriter io.Writer
)

// init sets the global logging level for zerolog to ErrorLevel by default
func init() {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	logWriter = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
}

// ConfigureGlobalLogging configures the global logging settings for the application.
func ConfigureGlobalLogging(levelStr string) error {
	level := parseLogLevel(levelStr)
	zerolog.SetGlobalLevel(level)

	w := getLogWriter()

	logContext := zerolog.New(w).With().Timestamp()
	if level <= zerolog.DebugLevel {
		logContext = logContext.Caller()
	}

	log.Logger = logContext.Logger().Level(level)
	zerolog.DefaultContextLogger = &log.Logger

	stdLog.SetFlags(stdLog.Lshortfile | stdLog.LstdFlags)
	stdLog.SetOutput(WithLevelOverride(log.Logger, zerolog.DebugLevel))

	return nil
}

// parseLogLevel converts a string log level to zerolog.Level
func parseLogLevel(levelString string) zerolog.Level {
	if levelString == "" {
		levelString = "error"
	}

	level, err := zerolog.ParseLevel(strings.ToLower(levelString))
	if err != nil {
		log.Error().Err(err).
			Str("logLevel", levelString).
			Msg("Invalid log level provided. Defaulting to error level.")
		return zerolog.ErrorLevel
	}
	return level
}

// getLogWriter returns the configured log writer
func getLogWriter() io.Writer {
	return logWriter
}

// SetLogWriter sets the global log writer
func SetLogWriter(w io.Writer) {
	logWriter = w
}

// WithLevelOverride is a helper function for standard log output
func WithLevelOverride(logger zerolog.Logger, level zerolog.Level) io.Writer {
	return logger.Level(level)
}
