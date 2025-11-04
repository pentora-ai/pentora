package fingerprint

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeTelemetry(t *testing.T) {
	t.Run("analyze basic telemetry file", func(t *testing.T) {
		// Create test telemetry file
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write test events
		events := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","target":"192.168.1.1","port":22,"protocol":"ssh","product":"OpenSSH","vendor":"OpenBSD","version":"8.2p1","confidence":0.95,"match_type":"success","resolver_name":"static","rule_id":"ssh.openssh"}`,
			`{"timestamp":"2024-01-01T10:01:00Z","target":"192.168.1.2","port":80,"protocol":"http","product":"Apache","vendor":"Apache","version":"2.4.41","confidence":0.85,"match_type":"success","resolver_name":"static","rule_id":"http.apache"}`,
			`{"timestamp":"2024-01-01T10:02:00Z","target":"192.168.1.3","port":443,"protocol":"https","confidence":0.0,"match_type":"no_match","resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:03:00Z","target":"192.168.1.4","port":22,"protocol":"ssh","confidence":0.0,"match_type":"rejected","resolver_name":"static","rule_id":"ssh.weak","rejection_reason":"hard_exclude_pattern"}`,
		}
		for _, event := range events {
			_, err := tmpFile.WriteString(event + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		// Analyze telemetry
		stats, err := AnalyzeTelemetry(tmpFile.Name(), nil)
		require.NoError(t, err)
		require.NotNil(t, stats)

		// Verify overall statistics
		require.Equal(t, 4, stats.TotalEvents)
		require.Equal(t, 2, stats.SuccessfulMatches)
		require.Equal(t, 1, stats.NoMatches)
		require.Equal(t, 1, stats.Rejections)
		require.InDelta(t, 0.50, stats.SuccessRate, 0.01)

		// Verify confidence statistics
		require.InDelta(t, 0.85, stats.ConfidenceStats.Min, 0.01)
		require.InDelta(t, 0.95, stats.ConfidenceStats.Max, 0.01)
		require.InDelta(t, 0.90, stats.ConfidenceStats.Average, 0.01)
		require.InDelta(t, 0.90, stats.ConfidenceStats.Median, 0.01)

		// Verify protocol stats
		require.Len(t, stats.ProtocolStats, 3)
		require.Contains(t, stats.ProtocolStats, "ssh")
		require.Contains(t, stats.ProtocolStats, "http")
		require.Contains(t, stats.ProtocolStats, "https")

		sshStats := stats.ProtocolStats["ssh"]
		require.Equal(t, 2, sshStats.TotalEvents)
		require.Equal(t, 1, sshStats.SuccessfulMatches)
		require.Equal(t, 0, sshStats.NoMatches)
		require.Equal(t, 1, sshStats.Rejections)
		require.InDelta(t, 0.95, sshStats.AvgConfidence, 0.01)

		// Verify top products
		require.Len(t, stats.TopProducts, 2)
		require.Equal(t, "OpenSSH", stats.TopProducts[0].Product)
		require.Equal(t, "OpenBSD", stats.TopProducts[0].Vendor)
		require.Equal(t, 1, stats.TopProducts[0].Count)

		// Verify rejection reasons
		require.Len(t, stats.RejectionReasons, 1)
		require.Equal(t, 1, stats.RejectionReasons["hard_exclude_pattern"])
	})

	t.Run("filter by protocol", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		events := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","port":22,"protocol":"ssh","product":"OpenSSH","confidence":0.95,"match_type":"success","resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:01:00Z","port":80,"protocol":"http","product":"Apache","confidence":0.85,"match_type":"success","resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:02:00Z","port":22,"protocol":"ssh","product":"Dropbear","confidence":0.80,"match_type":"success","resolver_name":"static"}`,
		}
		for _, event := range events {
			_, err := tmpFile.WriteString(event + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		filter := &StatsFilter{Protocol: "ssh"}
		stats, err := AnalyzeTelemetry(tmpFile.Name(), filter)
		require.NoError(t, err)

		require.Equal(t, 2, stats.TotalEvents)
		require.Equal(t, 2, stats.SuccessfulMatches)
		require.Len(t, stats.ProtocolStats, 1)
		require.Contains(t, stats.ProtocolStats, "ssh")
	})

	t.Run("filter by time range", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		events := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","port":22,"protocol":"ssh","match_type":"success","confidence":0.95,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-02T10:00:00Z","port":80,"protocol":"http","match_type":"success","confidence":0.85,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-03T10:00:00Z","port":443,"protocol":"https","match_type":"success","confidence":0.90,"resolver_name":"static"}`,
		}
		for _, event := range events {
			_, err := tmpFile.WriteString(event + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		since, _ := time.Parse(time.RFC3339, "2024-01-02T00:00:00Z")
		filter := &StatsFilter{Since: &since}
		stats, err := AnalyzeTelemetry(tmpFile.Name(), filter)
		require.NoError(t, err)

		require.Equal(t, 2, stats.TotalEvents) // Only Jan 2 and Jan 3
		require.Equal(t, 2, stats.SuccessfulMatches)
	})

	t.Run("limit top products", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Create 15 different products
		for i := 1; i <= 15; i++ {
			event := `{"timestamp":"2024-01-01T10:00:00Z","port":80,"protocol":"http","product":"Product` + string(rune(48+i)) + `","confidence":0.90,"match_type":"success","resolver_name":"static"}` + "\n"
			_, err := tmpFile.WriteString(event)
			require.NoError(t, err)
		}
		tmpFile.Close()

		filter := &StatsFilter{TopN: 5}
		stats, err := AnalyzeTelemetry(tmpFile.Name(), filter)
		require.NoError(t, err)

		require.Equal(t, 15, stats.TotalEvents)
		require.Len(t, stats.TopProducts, 5) // Limited to top 5
	})

	t.Run("empty telemetry file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		stats, err := AnalyzeTelemetry(tmpFile.Name(), nil)
		require.NoError(t, err)
		require.NotNil(t, stats)
		require.Equal(t, 0, stats.TotalEvents)
		require.Equal(t, 0, stats.SuccessfulMatches)
		require.Empty(t, stats.ProtocolStats)
		require.Empty(t, stats.TopProducts)
	})

	t.Run("file not found", func(t *testing.T) {
		stats, err := AnalyzeTelemetry("/nonexistent/file.jsonl", nil)
		require.Error(t, err)
		require.Nil(t, stats)
		require.Contains(t, err.Error(), "failed to open telemetry file")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(`{"invalid json content}`)
		require.NoError(t, err)
		tmpFile.Close()

		stats, err := AnalyzeTelemetry(tmpFile.Name(), nil)
		require.Error(t, err)
		require.Nil(t, stats)
		require.Contains(t, err.Error(), "failed to parse line")
	})

	t.Run("skip empty lines", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		events := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","port":22,"protocol":"ssh","match_type":"success","confidence":0.95,"resolver_name":"static"}`,
			``, // Empty line
			`{"timestamp":"2024-01-01T10:01:00Z","port":80,"protocol":"http","match_type":"success","confidence":0.85,"resolver_name":"static"}`,
		}
		for _, event := range events {
			_, err := tmpFile.WriteString(event + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		stats, err := AnalyzeTelemetry(tmpFile.Name(), nil)
		require.NoError(t, err)
		require.Equal(t, 2, stats.TotalEvents) // Empty line skipped
	})

	t.Run("multiple rejection reasons", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		events := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","port":22,"protocol":"ssh","match_type":"rejected","confidence":0.0,"resolver_name":"static","rejection_reason":"hard_exclude_pattern"}`,
			`{"timestamp":"2024-01-01T10:01:00Z","port":80,"protocol":"http","match_type":"rejected","confidence":0.0,"resolver_name":"static","rejection_reason":"confidence_below_threshold"}`,
			`{"timestamp":"2024-01-01T10:02:00Z","port":443,"protocol":"https","match_type":"rejected","confidence":0.0,"resolver_name":"static","rejection_reason":"hard_exclude_pattern"}`,
		}
		for _, event := range events {
			_, err := tmpFile.WriteString(event + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		stats, err := AnalyzeTelemetry(tmpFile.Name(), nil)
		require.NoError(t, err)

		require.Equal(t, 3, stats.Rejections)
		require.Len(t, stats.RejectionReasons, 2)
		require.Equal(t, 2, stats.RejectionReasons["hard_exclude_pattern"])
		require.Equal(t, 1, stats.RejectionReasons["confidence_below_threshold"])
	})

	t.Run("confidence distribution with odd number of values", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		events := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","port":22,"protocol":"ssh","match_type":"success","confidence":0.70,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:01:00Z","port":80,"protocol":"http","match_type":"success","confidence":0.80,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:02:00Z","port":443,"protocol":"https","match_type":"success","confidence":0.90,"resolver_name":"static"}`,
		}
		for _, event := range events {
			_, err := tmpFile.WriteString(event + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		stats, err := AnalyzeTelemetry(tmpFile.Name(), nil)
		require.NoError(t, err)

		require.InDelta(t, 0.80, stats.ConfidenceStats.Median, 0.01) // Middle value (odd count)
	})

	t.Run("confidence distribution with even number of values", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		events := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","port":22,"protocol":"ssh","match_type":"success","confidence":0.70,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:01:00Z","port":80,"protocol":"http","match_type":"success","confidence":0.80,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:02:00Z","port":443,"protocol":"https","match_type":"success","confidence":0.85,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:03:00Z","port":3306,"protocol":"mysql","match_type":"success","confidence":0.95,"resolver_name":"static"}`,
		}
		for _, event := range events {
			_, err := tmpFile.WriteString(event + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		stats, err := AnalyzeTelemetry(tmpFile.Name(), nil)
		require.NoError(t, err)

		require.InDelta(t, 0.825, stats.ConfidenceStats.Median, 0.01) // Average of middle two values
	})

	t.Run("product count with vendor", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-telemetry-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		events := []string{
			`{"timestamp":"2024-01-01T10:00:00Z","port":22,"protocol":"ssh","product":"OpenSSH","vendor":"OpenBSD","match_type":"success","confidence":0.95,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:01:00Z","port":22,"protocol":"ssh","product":"OpenSSH","vendor":"OpenBSD","match_type":"success","confidence":0.90,"resolver_name":"static"}`,
			`{"timestamp":"2024-01-01T10:02:00Z","port":22,"protocol":"ssh","product":"Dropbear","match_type":"success","confidence":0.85,"resolver_name":"static"}`,
		}
		for _, event := range events {
			_, err := tmpFile.WriteString(event + "\n")
			require.NoError(t, err)
		}
		tmpFile.Close()

		stats, err := AnalyzeTelemetry(tmpFile.Name(), nil)
		require.NoError(t, err)

		require.Len(t, stats.TopProducts, 2)

		// First product should be OpenBSD/OpenSSH with 2 detections
		require.Equal(t, "OpenSSH", stats.TopProducts[0].Product)
		require.Equal(t, "OpenBSD", stats.TopProducts[0].Vendor)
		require.Equal(t, 2, stats.TopProducts[0].Count)

		// Second product should be Dropbear with 1 detection
		require.Equal(t, "Dropbear", stats.TopProducts[1].Product)
		require.Equal(t, "", stats.TopProducts[1].Vendor)
		require.Equal(t, 1, stats.TopProducts[1].Count)
	})
}
