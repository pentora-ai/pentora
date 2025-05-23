package logging

import (
	"bytes"
	stdLog "log"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestParseLogLevel(t *testing.T) {
	assert.Equal(t, zerolog.DebugLevel, parseLogLevel("debug"))
	assert.Equal(t, zerolog.InfoLevel, parseLogLevel("info"))
	assert.Equal(t, zerolog.WarnLevel, parseLogLevel("warn"))
	assert.Equal(t, zerolog.ErrorLevel, parseLogLevel("error"))
	assert.Equal(t, zerolog.ErrorLevel, parseLogLevel("invalid")) // fallback
	assert.Equal(t, zerolog.ErrorLevel, parseLogLevel(""))        // default
}

func TestSetAndGetLogWriter(t *testing.T) {
	var buf bytes.Buffer
	SetLogWriter(&buf)
	assert.Equal(t, &buf, getLogWriter())
}

func TestConfigureGlobalLogging(t *testing.T) {
	var buf bytes.Buffer
	SetLogWriter(&buf)
	err := ConfigureGlobalLogging("debug")
	assert.NoError(t, err)
	log.Debug().Msg("test debug message")
	assert.Contains(t, buf.String(), "test debug message")
}

func TestStdLogWriterIntegration(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)
	writer := &stdLogWriter{logger: logger}

	// Simulate stdlog output
	stdLogMsg := "2025/05/23 14:40:15 version.go:35: Pentora version: dev\n"
	n, err := writer.Write([]byte(stdLogMsg))
	assert.Equal(t, len(stdLogMsg), n)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Pentora version: dev")
	assert.Contains(t, buf.String(), "version.go:35")
}

func TestStdLogWriterFallback(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)
	writer := &stdLogWriter{logger: logger}

	// Malformed stdlog output
	msg := "not a stdlog format\n"
	n, err := writer.Write([]byte(msg))
	assert.Equal(t, len(msg), n)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "not a stdlog format")
}

func TestLevelOverrideHook(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel)
	hook := NewLevelOverrideHook(zerolog.InfoLevel, zerolog.WarnLevel)
	logger = logger.Hook(hook)

	logger.WithLevel(zerolog.NoLevel).Msg("no level event")
	logger.Info().Msg("info event")
	logger.Debug().Msg("debug event") // should not appear

	out := buf.String()
	assert.Contains(t, out, "no level event")
	assert.Contains(t, out, "info event")
	assert.NotContains(t, out, "debug event")
}

func TestWithLevelOverride(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel)
	logger = WithLevelOverride(logger, zerolog.WarnLevel)
	logger.WithLevel(zerolog.NoLevel).Msg("should be warn level")
	assert.Contains(t, buf.String(), "should be warn level")
}

func TestLazyMessage(t *testing.T) {
	msgFunc := LazyMessage("foo", 123, "bar")
	assert.Equal(t, "foo123bar", msgFunc())
}

func TestConfigureGlobalLoggingStdLogRedirect(t *testing.T) {
	var buf bytes.Buffer
	SetLogWriter(&buf)
	_ = ConfigureGlobalLogging("debug")

	stdLog.Print("2025/05/23 14:40:15 version.go:35: redirected stdlog message")
	assert.Contains(t, buf.String(), "redirected stdlog message")
}

func TestSetLogWriterConcurrency(t *testing.T) {
	// This is a basic concurrency test for SetLogWriter/getLogWriter
	var buf1, buf2 bytes.Buffer
	SetLogWriter(&buf1)
	assert.Equal(t, &buf1, getLogWriter())
	SetLogWriter(&buf2)
	assert.Equal(t, &buf2, getLogWriter())
}

func TestStdLogWriterWriteHandlesTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)
	writer := &stdLogWriter{logger: logger}

	msg := "2025/05/23 14:40:15 version.go:35: message with newline\n"
	_, err := writer.Write([]byte(msg))
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "message with newline")
}

func TestStdLogWriterWriteHandlesNoTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)
	writer := &stdLogWriter{logger: logger}

	msg := "2025/05/23 14:40:15 version.go:35: message without newline"
	_, err := writer.Write([]byte(msg))
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "message without newline")
}
