package logging

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test-component", zerolog.InfoLevel)

	// Logger should be configured with component field
	require.NotNil(t, logger)
}

func TestNewLoggerWithWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("test", zerolog.DebugLevel, &buf)

	logger.Debug().Msg("test debug message")
	assert.Contains(t, buf.String(), "test debug message")
	assert.Contains(t, buf.String(), `"component":"test"`)
	assert.Contains(t, buf.String(), `"level":"debug"`)
}

func TestNewLoggerLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("test", zerolog.InfoLevel, &buf)

	// Debug should not appear (below info level)
	logger.Debug().Msg("debug message")
	assert.NotContains(t, buf.String(), "debug message")

	// Info should appear
	logger.Info().Msg("info message")
	assert.Contains(t, buf.String(), "info message")

	// Warn should appear
	logger.Warn().Msg("warn message")
	assert.Contains(t, buf.String(), "warn message")
}

func TestConfigureGlobal(t *testing.T) {
	// This test verifies ConfigureGlobal sets up global logger
	// Note: This modifies global state, so it's isolated
	ConfigureGlobal(zerolog.DebugLevel)

	// Global level should be set
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}

func TestNewLoggerComponentField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLoggerWithWriter("my-component", zerolog.InfoLevel, &buf)

	logger.Info().Msg("test message")
	output := buf.String()

	assert.Contains(t, output, `"component":"my-component"`)
	assert.Contains(t, output, "test message")
}

func TestNewLoggerMultipleInstances(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	logger1 := NewLoggerWithWriter("component-1", zerolog.InfoLevel, &buf1)
	logger2 := NewLoggerWithWriter("component-2", zerolog.WarnLevel, &buf2)

	logger1.Info().Msg("from logger 1")
	logger2.Warn().Msg("from logger 2")

	assert.Contains(t, buf1.String(), `"component":"component-1"`)
	assert.Contains(t, buf1.String(), "from logger 1")

	assert.Contains(t, buf2.String(), `"component":"component-2"`)
	assert.Contains(t, buf2.String(), "from logger 2")
}
