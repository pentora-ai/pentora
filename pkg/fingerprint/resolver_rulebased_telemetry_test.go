package fingerprint

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuleBasedResolver_TelemetryIntegration(t *testing.T) {
	t.Run("logs successful match", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		telemetry, err := NewTelemetryWriter(tmpFile.Name())
		require.NoError(t, err)
		defer telemetry.Close()

		rules := []StaticRule{
			{
				ID:              "test.ssh",
				Protocol:        "ssh",
				Product:         "OpenSSH",
				Vendor:          "OpenBSD",
				Match:           "openssh",
				PatternStrength: 0.90,
			},
		}

		resolver := NewRuleBasedResolver(rules)
		resolver.SetTelemetry(telemetry)

		input := Input{
			Port:     22,
			Protocol: "ssh",
			Banner:   "SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5",
		}

		result, err := resolver.Resolve(context.Background(), input)
		require.NoError(t, err)
		require.Equal(t, "OpenSSH", result.Product)

		// Read telemetry event
		data, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		var event DetectionEvent
		err = json.Unmarshal(data, &event)
		require.NoError(t, err)

		require.Equal(t, 22, event.Port)
		require.Equal(t, "ssh", event.Protocol)
		require.Equal(t, "OpenSSH", event.Product)
		require.Equal(t, "success", event.MatchType)
		require.Equal(t, "static", event.ResolverName)
		require.Equal(t, "test.ssh", event.RuleID)
	})

	t.Run("logs no match", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		telemetry, err := NewTelemetryWriter(tmpFile.Name())
		require.NoError(t, err)
		defer telemetry.Close()

		rules := []StaticRule{
			{
				ID:              "test.ssh",
				Protocol:        "ssh",
				Product:         "OpenSSH",
				Match:           "openssh",
				PatternStrength: 0.90,
			},
		}

		resolver := NewRuleBasedResolver(rules)
		resolver.SetTelemetry(telemetry)

		input := Input{
			Port:     22,
			Protocol: "ssh",
			Banner:   "SSH-2.0-UnknownSSH_1.0",
		}

		_, err = resolver.Resolve(context.Background(), input)
		require.Error(t, err)

		// Read telemetry event
		data, err := os.ReadFile(tmpFile.Name())
		require.NoError(t, err)

		var event DetectionEvent
		err = json.Unmarshal(data, &event)
		require.NoError(t, err)

		require.Equal(t, 22, event.Port)
		require.Equal(t, "ssh", event.Protocol)
		require.Equal(t, "no_match", event.MatchType)
		require.Equal(t, "static", event.ResolverName)
	})

	t.Run("logs hard exclude rejection", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		telemetry, err := NewTelemetryWriter(tmpFile.Name())
		require.NoError(t, err)
		defer telemetry.Close()

		rules := []StaticRule{
			{
				ID:              "test.http.apache",
				Protocol:        "http",
				Product:         "Apache",
				Match:           "apache",
				ExcludePatterns: []string{"nginx"},
				PatternStrength: 0.90,
			},
		}

		resolver := NewRuleBasedResolver(rules)
		resolver.SetTelemetry(telemetry)

		input := Input{
			Port:     80,
			Protocol: "http",
			Banner:   "Server: Apache/2.4.41 (running on nginx)",
		}

		_, err = resolver.Resolve(context.Background(), input)
		require.Error(t, err)

		// Read telemetry events (should have rejection + no_match)
		file, err := os.Open(tmpFile.Name())
		require.NoError(t, err)
		defer file.Close()

		decoder := json.NewDecoder(file)

		// First event: rejection
		var event1 DetectionEvent
		require.True(t, decoder.More())
		err = decoder.Decode(&event1)
		require.NoError(t, err)
		require.Equal(t, "rejected", event1.MatchType)
		require.Equal(t, "hard_exclude_pattern", event1.RejectionReason)
		require.Equal(t, "test.http.apache", event1.RuleID)

		// Second event: no_match
		var event2 DetectionEvent
		require.True(t, decoder.More())
		err = decoder.Decode(&event2)
		require.NoError(t, err)
		require.Equal(t, "no_match", event2.MatchType)
	})

	t.Run("logs confidence threshold rejection", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		telemetry, err := NewTelemetryWriter(tmpFile.Name())
		require.NoError(t, err)
		defer telemetry.Close()

		rules := []StaticRule{
			{
				ID:                  "test.http.apache",
				Protocol:            "http",
				Product:             "Apache",
				Match:               "apache",
				PatternStrength:     0.60,
				SoftExcludePatterns: []string{"error", "unavailable"},
			},
		}

		resolver := NewRuleBasedResolver(rules)
		resolver.SetTelemetry(telemetry)

		input := Input{
			Port:     80,
			Protocol: "http",
			Banner:   "Server: Apache/2.4.41 (error unavailable)",
		}

		_, err = resolver.Resolve(context.Background(), input)
		require.Error(t, err)

		// Read telemetry events
		file, err := os.Open(tmpFile.Name())
		require.NoError(t, err)
		defer file.Close()

		decoder := json.NewDecoder(file)

		// First event: confidence rejection
		var event1 DetectionEvent
		require.True(t, decoder.More())
		err = decoder.Decode(&event1)
		require.NoError(t, err)
		require.Equal(t, "rejected", event1.MatchType)
		require.Equal(t, "confidence_below_threshold", event1.RejectionReason)
		require.Equal(t, "test.http.apache", event1.RuleID)

		// Second event: no_match
		var event2 DetectionEvent
		require.True(t, decoder.More())
		err = decoder.Decode(&event2)
		require.NoError(t, err)
		require.Equal(t, "no_match", event2.MatchType)
	})

	t.Run("works without telemetry", func(t *testing.T) {
		rules := []StaticRule{
			{
				ID:              "test.ssh",
				Protocol:        "ssh",
				Product:         "OpenSSH",
				Match:           "openssh",
				PatternStrength: 0.90,
			},
		}

		resolver := NewRuleBasedResolver(rules)
		// No telemetry set

		input := Input{
			Port:     22,
			Protocol: "ssh",
			Banner:   "SSH-2.0-OpenSSH_8.2p1",
		}

		result, err := resolver.Resolve(context.Background(), input)
		require.NoError(t, err)
		require.Equal(t, "OpenSSH", result.Product)
	})
}
