package fingerprint

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// TelemetryStats represents aggregated statistics from telemetry data.
type TelemetryStats struct {
	// Overall statistics
	TotalEvents       int     `json:"total_events"`
	SuccessfulMatches int     `json:"successful_matches"`
	NoMatches         int     `json:"no_matches"`
	Rejections        int     `json:"rejections"`
	SuccessRate       float64 `json:"success_rate"`

	// Protocol breakdown
	ProtocolStats map[string]*ProtocolStat `json:"protocol_stats"`

	// Top detected products
	TopProducts []ProductCount `json:"top_products"`

	// Rejection reasons
	RejectionReasons map[string]int `json:"rejection_reasons"`

	// Confidence distribution
	ConfidenceStats ConfidenceStat `json:"confidence_stats"`

	// Time range
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
}

// ProtocolStat represents statistics for a specific protocol.
type ProtocolStat struct {
	TotalEvents       int     `json:"total_events"`
	SuccessfulMatches int     `json:"successful_matches"`
	NoMatches         int     `json:"no_matches"`
	Rejections        int     `json:"rejections"`
	AvgConfidence     float64 `json:"avg_confidence"`
}

// ProductCount represents a product and its detection count.
type ProductCount struct {
	Product string `json:"product"`
	Vendor  string `json:"vendor,omitempty"`
	Count   int    `json:"count"`
}

// ConfidenceStat represents confidence score distribution.
type ConfidenceStat struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Average float64 `json:"average"`
	Median  float64 `json:"median"`
}

// StatsFilter provides filtering options for telemetry analysis.
type StatsFilter struct {
	Protocol  string     // Filter by protocol (e.g., "ssh", "http")
	Since     *time.Time // Start time filter
	Until     *time.Time // End time filter
	TopN      int        // Number of top products to include (default: 10)
	MinEvents int        // Minimum events to include protocol in stats
}

// AnalyzeTelemetry reads a JSONL telemetry file and computes aggregate statistics.
//
//nolint:gocyclo,funlen // Stats aggregation inherently complex, refactor planned for later
func AnalyzeTelemetry(filePath string, filter *StatsFilter) (*TelemetryStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open telemetry file: %w", err)
	}
	defer func() { _ = file.Close() }()

	stats := &TelemetryStats{
		ProtocolStats:    make(map[string]*ProtocolStat),
		RejectionReasons: make(map[string]int),
	}

	// Apply defaults
	if filter == nil {
		filter = &StatsFilter{TopN: 10}
	}
	if filter.TopN == 0 {
		filter.TopN = 10
	}

	productCounts := make(map[string]*ProductCount)
	confidenceScores := []float64{}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if line == "" {
			continue // Skip empty lines
		}

		var event DetectionEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}

		// Apply filters
		if filter.Protocol != "" && event.Protocol != filter.Protocol {
			continue
		}
		if filter.Since != nil && event.Timestamp.Before(*filter.Since) {
			continue
		}
		if filter.Until != nil && event.Timestamp.After(*filter.Until) {
			continue
		}

		// Update time range
		if stats.StartTime.IsZero() || event.Timestamp.Before(stats.StartTime) {
			stats.StartTime = event.Timestamp
		}
		if stats.EndTime.IsZero() || event.Timestamp.After(stats.EndTime) {
			stats.EndTime = event.Timestamp
		}

		// Overall statistics
		stats.TotalEvents++
		switch event.MatchType {
		case "success":
			stats.SuccessfulMatches++
			confidenceScores = append(confidenceScores, event.Confidence)

			// Track products
			key := event.Product
			if event.Vendor != "" {
				key = event.Vendor + "/" + event.Product
			}
			if pc, exists := productCounts[key]; exists {
				pc.Count++
			} else {
				productCounts[key] = &ProductCount{
					Product: event.Product,
					Vendor:  event.Vendor,
					Count:   1,
				}
			}
		case "no_match":
			stats.NoMatches++
		case "rejected":
			stats.Rejections++
			if event.RejectionReason != "" {
				stats.RejectionReasons[event.RejectionReason]++
			}
		}

		// Protocol statistics
		if _, exists := stats.ProtocolStats[event.Protocol]; !exists {
			stats.ProtocolStats[event.Protocol] = &ProtocolStat{}
		}
		protocolStat := stats.ProtocolStats[event.Protocol]
		protocolStat.TotalEvents++

		switch event.MatchType {
		case "success":
			protocolStat.SuccessfulMatches++
		case "no_match":
			protocolStat.NoMatches++
		case "rejected":
			protocolStat.Rejections++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading telemetry file: %w", err)
	}

	// Calculate success rate
	if stats.TotalEvents > 0 {
		stats.SuccessRate = float64(stats.SuccessfulMatches) / float64(stats.TotalEvents)
	}

	// Calculate protocol average confidences
	for protocol, protocolStat := range stats.ProtocolStats {
		if protocolStat.SuccessfulMatches > 0 {
			// Recalculate from events for protocol-specific average
			_, _ = file.Seek(0, 0) // Reset file pointer
			scanner = bufio.NewScanner(file)
			protocolConfidences := []float64{}

			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}

				var event DetectionEvent
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					continue // Skip malformed lines
				}

				if event.Protocol == protocol && event.MatchType == "success" {
					protocolConfidences = append(protocolConfidences, event.Confidence)
				}
			}

			if len(protocolConfidences) > 0 {
				sum := 0.0
				for _, conf := range protocolConfidences {
					sum += conf
				}
				protocolStat.AvgConfidence = sum / float64(len(protocolConfidences))
			}
		}
	}

	// Calculate confidence statistics
	if len(confidenceScores) > 0 {
		sort.Float64s(confidenceScores)
		stats.ConfidenceStats.Min = confidenceScores[0]
		stats.ConfidenceStats.Max = confidenceScores[len(confidenceScores)-1]

		sum := 0.0
		for _, score := range confidenceScores {
			sum += score
		}
		stats.ConfidenceStats.Average = sum / float64(len(confidenceScores))

		// Calculate median
		mid := len(confidenceScores) / 2
		if len(confidenceScores)%2 == 0 {
			stats.ConfidenceStats.Median = (confidenceScores[mid-1] + confidenceScores[mid]) / 2
		} else {
			stats.ConfidenceStats.Median = confidenceScores[mid]
		}
	}

	// Sort and limit top products
	productList := make([]ProductCount, 0, len(productCounts))
	for _, pc := range productCounts {
		productList = append(productList, *pc)
	}
	sort.Slice(productList, func(i, j int) bool {
		return productList[i].Count > productList[j].Count
	})

	if len(productList) > filter.TopN {
		productList = productList[:filter.TopN]
	}
	stats.TopProducts = productList

	return stats, nil
}
