// pkg/core/logger.go
package core

import (
	"io"
	stdlog "log"
	"os"
	"strings"
	"time"

	"github.com/pentora-ai/pentora/pkg/config/static"
	"github.com/pentora-ai/pentora/pkg/logs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sirupsen/logrus"
)

// init initializes the logger by setting the global log level to ErrorLevel.
// This ensures that any logs generated before the logger setup are suppressed.
func init() {
	// hide the first logs before the setup of the logger.
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
}

// setupLogger configures the logging system for the application.
//
// This function sets up the logger with the specified static configuration,
// including log format, log level, and output destinations. It also integrates
// with third-party logging libraries like logrus and the standard library's
// log package to ensure consistent logging behavior across the application.
//
// Parameters:
//   - staticConfiguration: A pointer to a static.Configuration object that
//     contains the logging configuration settings.
//
// Returns:
//   - An error if there is an issue during the logger setup, otherwise nil.
func SetupLogger(staticConfiguration *static.Configuration) error {

	// configure log format
	w := getLogWriter(staticConfiguration)

	// configure log level
	logLevel := getLogLevel(staticConfiguration)
	zerolog.SetGlobalLevel(logLevel)

	// create logger
	logCtx := zerolog.New(w).With().Timestamp()
	if logLevel <= zerolog.DebugLevel {
		logCtx = logCtx.Caller()
	}

	log.Logger = logCtx.Logger().Level(logLevel)

	zerolog.DefaultContextLogger = &log.Logger

	// Global logrus replacement (related to lib like go-rancher-metadata, docker, etc.)
	logrus.StandardLogger().Out = logs.NoLevel(log.Logger, zerolog.DebugLevel)

	// configure default standard log.
	stdlog.SetFlags(stdlog.Lshortfile | stdlog.LstdFlags)
	stdlog.SetOutput(logs.NoLevel(log.Logger, zerolog.DebugLevel))

	return nil
}

// getLogWriter returns an io.Writer for logging purposes based on the provided
// static configuration. By default, it writes logs to os.Stdout. If the log
// format specified in the configuration is not "json", it uses a zerolog.ConsoleWriter
// with the specified time format and color settings.
//
// Parameters:
//   - staticConfiguration: A pointer to a static.Configuration struct that contains
//     the logging configuration, including format and color settings.
//
// Returns:
//   - An io.Writer configured for logging.
func getLogWriter(staticConfiguration *static.Configuration) io.Writer {
	var w io.Writer = os.Stdout

	if staticConfiguration.Log != nil && staticConfiguration.Log.Format != "json" {
		w = zerolog.ConsoleWriter{
			Out:        w,
			TimeFormat: time.RFC3339,
			NoColor:    staticConfiguration.Log != nil && staticConfiguration.Log.NoColor,
		}
	}

	return w
}

// getLogLevel determines the logging level based on the provided static configuration.
// If a valid log level is specified in the configuration, it is used. Otherwise, the default
// log level is set to "error". If an invalid log level is provided, an error is logged, and
// the log level is set to "error".
//
// Parameters:
//   - staticConfiguration: A pointer to a static.Configuration object that may contain
//     logging configuration details.
//
// Returns:
//   - zerolog.Level: The resolved logging level to be used.
func getLogLevel(staticConfiguration *static.Configuration) zerolog.Level {
	levelStr := "error"
	if staticConfiguration.Log != nil && staticConfiguration.Log.Level != "" {
		levelStr = strings.ToLower(staticConfiguration.Log.Level)
	}

	logLevel, err := zerolog.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		log.Error().Err(err).
			Str("logLevel", levelStr).
			Msg("Unspecified or invalid log level, setting the level to default (ERROR)...")

		logLevel = zerolog.ErrorLevel
	}

	return logLevel
}
