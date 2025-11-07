package fingerprint

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// DetectionEvent represents a single service detection event for telemetry.
type DetectionEvent struct {
	Timestamp       time.Time `json:"timestamp"`
	Target          string    `json:"target"`
	Port            int       `json:"port"`
	Protocol        string    `json:"protocol"`
	Product         string    `json:"product,omitempty"`
	Vendor          string    `json:"vendor,omitempty"`
	Version         string    `json:"version,omitempty"`
	Confidence      float64   `json:"confidence"`
	MatchType       string    `json:"match_type"`    // "success", "no_match", "rejected"
	ResolverName    string    `json:"resolver_name"` // "static", "ml", "heuristic", etc.
	RuleID          string    `json:"rule_id,omitempty"`
	RejectionReason string    `json:"rejection_reason,omitempty"` // For anti-pattern rejections
}

// TelemetryWriter writes detection events to a JSONL file in a thread-safe manner.
type TelemetryWriter struct {
	filePath string
	file     *os.File
	encoder  *json.Encoder
	mu       sync.Mutex
	enabled  bool
}

// NewTelemetryWriter creates a new telemetry writer that appends to the specified file.
// If filePath is empty, the writer is disabled.
func NewTelemetryWriter(filePath string) (*TelemetryWriter, error) {
	if filePath == "" {
		return &TelemetryWriter{enabled: false}, nil
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open telemetry file: %w", err)
	}

	return &TelemetryWriter{
		filePath: filePath,
		file:     file,
		encoder:  json.NewEncoder(file),
		enabled:  true,
	}, nil
}

// Write writes a detection event to the telemetry file.
// This operation is thread-safe.
func (w *TelemetryWriter) Write(event DetectionEvent) error {
	if !w.enabled {
		return nil // Silently skip if disabled
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to write telemetry event: %w", err)
	}

	return nil
}

// WriteSuccess writes a successful detection event.
func (w *TelemetryWriter) WriteSuccess(target string, port int, protocol string, result Result, resolverName, ruleID string) error {
	event := DetectionEvent{
		Timestamp:    time.Now(),
		Target:       target,
		Port:         port,
		Protocol:     protocol,
		Product:      result.Product,
		Vendor:       result.Vendor,
		Version:      result.Version,
		Confidence:   result.Confidence,
		MatchType:    "success",
		ResolverName: resolverName,
		RuleID:       ruleID,
	}
	return w.Write(event)
}

// WriteNoMatch writes a no-match event when no rule matched the banner.
func (w *TelemetryWriter) WriteNoMatch(target string, port int, protocol, resolverName string) error {
	event := DetectionEvent{
		Timestamp:    time.Now(),
		Target:       target,
		Port:         port,
		Protocol:     protocol,
		Confidence:   0.0,
		MatchType:    "no_match",
		ResolverName: resolverName,
	}
	return w.Write(event)
}

// WriteRejected writes a rejection event when anti-patterns triggered.
func (w *TelemetryWriter) WriteRejected(target string, port int, protocol, reason, resolverName, ruleID string) error {
	event := DetectionEvent{
		Timestamp:       time.Now(),
		Target:          target,
		Port:            port,
		Protocol:        protocol,
		Confidence:      0.0,
		MatchType:       "rejected",
		ResolverName:    resolverName,
		RuleID:          ruleID,
		RejectionReason: reason,
	}
	return w.Write(event)
}

// Close closes the telemetry file.
func (w *TelemetryWriter) Close() error {
	if !w.enabled || w.file == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close telemetry file: %w", err)
	}

	w.file = nil
	return nil
}

// IsEnabled returns true if telemetry is enabled.
func (w *TelemetryWriter) IsEnabled() bool {
	return w.enabled
}
