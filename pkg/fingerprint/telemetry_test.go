package fingerprint

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewTelemetryWriter(t *testing.T) {
	t.Run("disabled when empty path", func(t *testing.T) {
		writer, err := NewTelemetryWriter("")
		require.NoError(t, err)
		require.NotNil(t, writer)
		require.False(t, writer.IsEnabled())
	})

	t.Run("creates file successfully", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		writer, err := NewTelemetryWriter(tmpFile.Name())
		require.NoError(t, err)
		require.NotNil(t, writer)
		require.True(t, writer.IsEnabled())

		err = writer.Close()
		require.NoError(t, err)
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		writer, err := NewTelemetryWriter("/nonexistent/directory/telemetry.jsonl")
		require.Error(t, err)
		require.Nil(t, writer)
	})
}

func TestTelemetryWriter_Write(t *testing.T) {
	t.Run("writes event to file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		writer, err := NewTelemetryWriter(tmpFile.Name())
		require.NoError(t, err)
		defer writer.Close()

		event := DetectionEvent{
			Timestamp:    time.Now(),
			Target:       "192.168.1.1",
			Port:         22,
			Protocol:     "ssh",
			Product:      "OpenSSH",
			Vendor:       "OpenBSD",
			Version:      "8.2p1",
			Confidence:   0.95,
			MatchType:    "success",
			ResolverName: "static",
			RuleID:       "ssh.openssh",
		}

		err = writer.Write(event)
		require.NoError(t, err)

		// Read back and verify
		data, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		var readEvent DetectionEvent
		err = json.Unmarshal(data, &readEvent)
		require.NoError(t, err)

		require.Equal(t, event.Target, readEvent.Target)
		require.Equal(t, event.Port, readEvent.Port)
		require.Equal(t, event.Protocol, readEvent.Protocol)
		require.Equal(t, event.Product, readEvent.Product)
		require.Equal(t, event.Confidence, readEvent.Confidence)
	})

	t.Run("skips write when disabled", func(t *testing.T) {
		writer, err := NewTelemetryWriter("")
		require.NoError(t, err)
		require.False(t, writer.IsEnabled())

		event := DetectionEvent{
			Timestamp: time.Now(),
			Target:    "192.168.1.1",
			Port:      22,
			Protocol:  "ssh",
		}

		err = writer.Write(event)
		require.NoError(t, err) // Should not error
	})

	t.Run("writes multiple events", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		writer, err := NewTelemetryWriter(tmpFile.Name())
		require.NoError(t, err)
		defer writer.Close()

		// Write multiple events
		for i := 0; i < 5; i++ {
			event := DetectionEvent{
				Timestamp:    time.Now(),
				Target:       "192.168.1.1",
				Port:         22 + i,
				Protocol:     "ssh",
				MatchType:    "success",
				ResolverName: "static",
			}
			err = writer.Write(event)
			require.NoError(t, err)
		}

		// Read back and count lines
		file, err := os.Open(tmpFile.Name())
		require.NoError(t, err)
		defer file.Close()

		lines := 0
		decoder := json.NewDecoder(file)
		for decoder.More() {
			var event DetectionEvent
			err := decoder.Decode(&event)
			require.NoError(t, err)
			lines++
		}

		require.Equal(t, 5, lines)
	})
}

func TestTelemetryWriter_WriteSuccess(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	writer, err := NewTelemetryWriter(tmpFile.Name())
	require.NoError(t, err)
	defer writer.Close()

	result := Result{
		Product:    "OpenSSH",
		Vendor:     "OpenBSD",
		Version:    "8.2p1",
		Confidence: 0.95,
	}

	err = writer.WriteSuccess("192.168.1.1", 22, "ssh", result, "static", "ssh.openssh")
	require.NoError(t, err)

	// Read back and verify
	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var event DetectionEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	require.Equal(t, "192.168.1.1", event.Target)
	require.Equal(t, 22, event.Port)
	require.Equal(t, "ssh", event.Protocol)
	require.Equal(t, "OpenSSH", event.Product)
	require.Equal(t, "OpenBSD", event.Vendor)
	require.Equal(t, "8.2p1", event.Version)
	require.Equal(t, 0.95, event.Confidence)
	require.Equal(t, "success", event.MatchType)
	require.Equal(t, "static", event.ResolverName)
	require.Equal(t, "ssh.openssh", event.RuleID)
}

func TestTelemetryWriter_WriteNoMatch(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	writer, err := NewTelemetryWriter(tmpFile.Name())
	require.NoError(t, err)
	defer writer.Close()

	err = writer.WriteNoMatch("192.168.1.1", 8080, "http", "static")
	require.NoError(t, err)

	// Read back and verify
	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var event DetectionEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	require.Equal(t, "192.168.1.1", event.Target)
	require.Equal(t, 8080, event.Port)
	require.Equal(t, "http", event.Protocol)
	require.Equal(t, 0.0, event.Confidence)
	require.Equal(t, "no_match", event.MatchType)
	require.Equal(t, "static", event.ResolverName)
}

func TestTelemetryWriter_WriteRejected(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	writer, err := NewTelemetryWriter(tmpFile.Name())
	require.NoError(t, err)
	defer writer.Close()

	err = writer.WriteRejected("192.168.1.1", 22, "ssh", "hard_exclude_pattern", "static", "ssh.openssh")
	require.NoError(t, err)

	// Read back and verify
	data, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var event DetectionEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err)

	require.Equal(t, "192.168.1.1", event.Target)
	require.Equal(t, 22, event.Port)
	require.Equal(t, "ssh", event.Protocol)
	require.Equal(t, 0.0, event.Confidence)
	require.Equal(t, "rejected", event.MatchType)
	require.Equal(t, "static", event.ResolverName)
	require.Equal(t, "ssh.openssh", event.RuleID)
	require.Equal(t, "hard_exclude_pattern", event.RejectionReason)
}

func TestTelemetryWriter_Close(t *testing.T) {
	t.Run("closes file successfully", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		writer, err := NewTelemetryWriter(tmpFile.Name())
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Verify file is closed by trying to write
		event := DetectionEvent{
			Timestamp: time.Now(),
			Target:    "192.168.1.1",
			Port:      22,
			Protocol:  "ssh",
		}
		err = writer.Write(event)
		require.Error(t, err) // Should error because file is closed
	})

	t.Run("safe to close when disabled", func(t *testing.T) {
		writer, err := NewTelemetryWriter("")
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err) // Should not error
	})
}

func TestTelemetryWriter_ThreadSafety(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	writer, err := NewTelemetryWriter(tmpFile.Name())
	require.NoError(t, err)
	defer writer.Close()

	// Write events concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(port int) {
			event := DetectionEvent{
				Timestamp:    time.Now(),
				Target:       "192.168.1.1",
				Port:         port,
				Protocol:     "ssh",
				MatchType:    "success",
				ResolverName: "static",
			}
			err := writer.Write(event)
			require.NoError(t, err)
			done <- true
		}(22 + i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all events were written
	file, err := os.Open(tmpFile.Name())
	require.NoError(t, err)
	defer file.Close()

	decoder := json.NewDecoder(file)
	count := 0
	for decoder.More() {
		var event DetectionEvent
		err := decoder.Decode(&event)
		require.NoError(t, err)
		count++
	}

	require.Equal(t, 10, count)
}
