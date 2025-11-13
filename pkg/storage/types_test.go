package storage

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestVulnCounts_Total(t *testing.T) {
	tests := []struct {
		name     string
		counts   VulnCounts
		expected int
	}{
		{
			name:     "all zeros",
			counts:   VulnCounts{},
			expected: 0,
		},
		{
			name: "only critical",
			counts: VulnCounts{
				Critical: 5,
			},
			expected: 5,
		},
		{
			name: "mixed counts",
			counts: VulnCounts{
				Critical: 2,
				High:     3,
				Medium:   5,
				Low:      10,
				Info:     1,
			},
			expected: 21,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.counts.Total()
			if got != tt.expected {
				t.Errorf("Total() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestDataType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		dataType DataType
		expected bool
	}{
		{"metadata", DataTypeMetadata, true},
		{"hosts", DataTypeHosts, true},
		{"services", DataTypeServices, true},
		{"vulnerabilities", DataTypeVulnerabilities, true},
		{"banners", DataTypeBanners, true},
		{"invalid", DataType("invalid.txt"), false},
		{"empty", DataType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dataType.IsValid()
			if got != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestDataType_String(t *testing.T) {
	tests := []struct {
		name     string
		dataType DataType
		expected string
	}{
		{"metadata", DataTypeMetadata, "metadata.json"},
		{"hosts", DataTypeHosts, "hosts.jsonl"},
		{"services", DataTypeServices, "services.jsonl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dataType.String()
			if got != tt.expected {
				t.Errorf("String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestScanStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   ScanStatus
		expected bool
	}{
		{"pending", StatusPending, true},
		{"running", StatusRunning, true},
		{"completed", StatusCompleted, true},
		{"failed", StatusFailed, true},
		{"canceled", StatusCancelled, true},
		{"invalid", ScanStatus("invalid"), false},
		{"empty", ScanStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsValid()
			if got != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestScanStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   ScanStatus
		expected bool
	}{
		{"pending - not terminal", StatusPending, false},
		{"running - not terminal", StatusRunning, false},
		{"completed - terminal", StatusCompleted, true},
		{"failed - terminal", StatusFailed, true},
		{"canceled - terminal", StatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsTerminal()
			if got != tt.expected {
				t.Errorf("IsTerminal() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestScanStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   ScanStatus
		expected string
	}{
		{"pending", StatusPending, "pending"},
		{"running", StatusRunning, "running"},
		{"completed", StatusCompleted, "completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.expected {
				t.Errorf("String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

// TestScanMetadata_ExtensionsNotSerialized verifies that Extensions field
// is excluded from JSON serialization (json:"-" tag enforcement).
//
// This test ensures CE LocalBackend never persists EE-specific metadata.
func TestScanMetadata_ExtensionsNotSerialized(t *testing.T) {
	metadata := &ScanMetadata{
		ID:        "scan-123",
		OrgID:     "org-456",
		UserID:    "user-789",
		Target:    "192.168.1.0/24",
		Status:    "completed",
		StartedAt: time.Now(),
		Extensions: map[string]any{
			"ee_license_tier": "enterprise",
			"ee_org_name":     "ACME Corp",
			"ee_audit_id":     "audit-xyz",
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonString := string(jsonData)

	// Verify Extensions data NOT present in JSON
	if strings.Contains(jsonString, "ee_license_tier") {
		t.Errorf("Extensions field leaked into JSON: found 'ee_license_tier'")
	}
	if strings.Contains(jsonString, "ee_org_name") {
		t.Errorf("Extensions field leaked into JSON: found 'ee_org_name'")
	}
	if strings.Contains(jsonString, "ee_audit_id") {
		t.Errorf("Extensions field leaked into JSON: found 'ee_audit_id'")
	}
	if strings.Contains(jsonString, "extensions") {
		t.Errorf("Extensions field name appeared in JSON")
	}

	// Verify base fields ARE present
	if !strings.Contains(jsonString, "scan-123") {
		t.Errorf("Base field 'id' missing from JSON")
	}
	if !strings.Contains(jsonString, "org-456") {
		t.Errorf("Base field 'org_id' missing from JSON")
	}
}

// TestScanMetadata_ExtensionsRoundTrip verifies that Extensions field
// is not affected by JSON round-trip (serialize → deserialize).
//
// After deserialization, Extensions should be nil (not persisted).
func TestScanMetadata_ExtensionsRoundTrip(t *testing.T) {
	original := &ScanMetadata{
		ID:     "scan-abc",
		OrgID:  "org-xyz",
		UserID: "user-123",
		Target: "10.0.0.1",
		Status: "running",
		Extensions: map[string]any{
			"ee_metadata": "should_not_persist",
		},
	}

	// Serialize
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Deserialize
	var restored ScanMetadata
	if err := json.Unmarshal(jsonData, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Extensions should be nil after round-trip (not persisted)
	if restored.Extensions != nil {
		t.Errorf("Extensions field should be nil after JSON round-trip, got: %v", restored.Extensions)
	}

	// Base fields should be preserved
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", restored.ID, original.ID)
	}
	if restored.OrgID != original.OrgID {
		t.Errorf("OrgID mismatch: got %q, want %q", restored.OrgID, original.OrgID)
	}
	if restored.Status != original.Status {
		t.Errorf("Status mismatch: got %q, want %q", restored.Status, original.Status)
	}
}

// TestScanFilter_ExtensionsIgnoredInCE verifies that Extensions field
// in ScanFilter is present but unused in CE logic.
//
// This test documents that CE code does not process Extensions.
func TestScanFilter_ExtensionsIgnoredInCE(t *testing.T) {
	// CE creates filter without Extensions
	filterCE := ScanFilter{
		Status: "completed",
		Limit:  10,
	}

	// EE might create filter with Extensions
	filterEE := ScanFilter{
		Status: "completed",
		Limit:  10,
		Extensions: map[string]any{
			"ee_license_tier": "enterprise",
		},
	}

	// Both filters should compile and be usable
	// (CE code ignores Extensions, EE code uses it)
	if filterCE.Status != "completed" {
		t.Errorf("CE filter Status incorrect")
	}
	if filterEE.Status != "completed" {
		t.Errorf("EE filter Status incorrect")
	}

	// Verify Extensions field exists but is optional
	if filterCE.Extensions != nil {
		t.Errorf("CE filter Extensions should be nil (not set)")
	}
	if filterEE.Extensions == nil {
		t.Errorf("EE filter Extensions should be set")
	}
}

// TestScanUpdates_ExtensionsNotSerialized verifies Extensions field
// in ScanUpdates is also excluded from JSON serialization.
func TestScanUpdates_ExtensionsNotSerialized(t *testing.T) {
	status := "completed"
	extensionsData := map[string]any{
		"ee_updated_by": "admin@example.com",
	}

	updates := &ScanUpdates{
		Status:     &status,
		Extensions: &extensionsData,
	}

	// Serialize
	jsonData, err := json.Marshal(updates)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonString := string(jsonData)

	// Verify Extensions NOT in JSON
	if strings.Contains(jsonString, "ee_updated_by") {
		t.Errorf("Extensions data leaked into JSON: found 'ee_updated_by'")
	}
	if strings.Contains(jsonString, "extensions") {
		t.Errorf("Extensions field name appeared in JSON")
	}

	// Verify Status IS in JSON
	if !strings.Contains(jsonString, "completed") {
		t.Errorf("Status field missing from JSON")
	}
}

// TestExtensions_CEBehaviorUnchanged verifies that adding Extensions field
// does not break existing CE functionality.
//
// This test ensures backward compatibility with existing CE code.
func TestExtensions_CEBehaviorUnchanged(t *testing.T) {
	// Create metadata WITHOUT Extensions (existing CE pattern)
	metadata := &ScanMetadata{
		ID:     "scan-old",
		OrgID:  "default",
		UserID: "local",
		Target: "192.168.1.1",
		Status: "completed",
	}

	// Serialize (CE LocalBackend pattern)
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Existing CE serialization broken: %v", err)
	}

	// Deserialize (CE LocalBackend read pattern)
	var restored ScanMetadata
	if err := json.Unmarshal(jsonData, &restored); err != nil {
		t.Fatalf("Existing CE deserialization broken: %v", err)
	}

	// Verify all base fields work as before
	if restored.ID != metadata.ID {
		t.Errorf("CE behavior changed: ID mismatch")
	}
	if restored.OrgID != metadata.OrgID {
		t.Errorf("CE behavior changed: OrgID mismatch")
	}
	if restored.Status != metadata.Status {
		t.Errorf("CE behavior changed: Status mismatch")
	}

	// Extensions should be nil (not set, not breaking anything)
	if restored.Extensions != nil {
		t.Errorf("Extensions should remain nil for CE-created metadata")
	}
}
