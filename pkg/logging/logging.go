// pkg/logging/logging.go
package logging

import (
	"fmt"
	"io"
	stdLog "log"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// logWriter stores the current log writer globally
	logWriter io.Writer
)

// stdLogWriter is a custom writer that reformats stdlog output to match zerolog's format
type stdLogWriter struct {
	logger zerolog.Logger
}

func (w *stdLogWriter) Write(p []byte) (n int, err error) {
	// Remove trailing newline if exists
	message := strings.TrimSuffix(string(p), "\n")

	// Parse the stdlog format (this is a simplified parser)
	// Example stdlog output: "2025/05/23 14:40:15 version.go:35: Pentora version: dev"
	parts := strings.SplitN(message, " ", 4)
	if len(parts) >= 4 {
		// Reformat the timestamp
		stdTime, err := time.Parse("2006/01/02 15:04:05", parts[0]+" "+parts[1])
		if err == nil {
			// Extract filename and line number (parts[2] is "version.go:35:")
			fileLine := strings.TrimSuffix(parts[2], ":")

			// Log with zerolog format
			w.logger.Debug().
				Str("file", fileLine).
				Time("time", stdTime).
				Msg(parts[3])
			return len(p), nil
		}
	}

	// Fallback if parsing fails
	w.logger.Debug().Msg(message)
	return len(p), nil
}

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

	// Configure stdlog to use our custom writer
	stdLog.SetFlags(0) // Disable stdlog's own prefixes
	stdLog.SetOutput(&stdLogWriter{logger: WithLevelOverride(log.Logger, zerolog.DebugLevel)})

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

// LevelOverrideHook provides functionality to override log levels
// and filter logs below a minimum severity level.
type LevelOverrideHook struct {
	minSeverity zerolog.Level // Minimum log level to keep
	targetLevel zerolog.Level // Level to assign to NoLevel events
}

// NewLevelOverrideHook creates a new LevelOverrideHook instance.
// minSeverity: Logs below this level will be discarded
// targetLevel: NoLevel events will be upgraded to this level
func NewLevelOverrideHook(minSeverity, targetLevel zerolog.Level) *LevelOverrideHook {
	return &LevelOverrideHook{
		minSeverity: minSeverity,
		targetLevel: targetLevel,
	}
}

// Run implements zerolog.Hook interface and performs the log level processing.
func (h LevelOverrideHook) Run(e *zerolog.Event, currentLevel zerolog.Level, _ string) {
	// Discard logs below our minimum severity
	if h.minSeverity > h.targetLevel {
		e.Discard()
		return
	}

	// Upgrade NoLevel events to our target level
	if currentLevel == zerolog.NoLevel {
		e.Str("level", h.targetLevel.String())
	}
}

// WithLevelOverride configures a logger to handle NoLevel events and level filtering.
func WithLevelOverride(logger zerolog.Logger, targetLevel zerolog.Level) zerolog.Logger {
	return logger.Hook(NewLevelOverrideHook(logger.GetLevel(), targetLevel))
}

// LazyMessage creates a closure for deferred message evaluation.
// Useful for expensive-to-compute log messages that should only be evaluated if needed.
func LazyMessage(args ...interface{}) func() string {
	return func() string { return fmt.Sprint(args...) }
}
