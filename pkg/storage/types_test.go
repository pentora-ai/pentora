package storage

import (
	"testing"
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
