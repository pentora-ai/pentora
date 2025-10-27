package storage

import "time"

// ScanMetadata contains metadata about a scan.
//
// This structure is stored in both OSS (metadata.json files) and
// Enterprise (scans table in PostgreSQL).
//
// The same structure is used for both editions to ensure compatibility.
type ScanMetadata struct {
	// ID is the unique identifier for the scan.
	// Format: UUID v4 or custom format.
	ID string `json:"id"`

	// OrgID identifies the organization that owns this scan.
	// OSS uses "default", Enterprise uses actual organization IDs.
	OrgID string `json:"org_id"`

	// UserID identifies the user who created the scan.
	// OSS uses "local", Enterprise uses actual user IDs.
	UserID string `json:"user_id"`

	// Target is the scan target specification.
	// Examples: "192.168.1.0/24", "example.com", "10.0.0.1-10.0.0.255"
	Target string `json:"target"`

	// Status indicates the current state of the scan.
	// Valid values: "pending", "running", "completed", "failed", "canceled"
	Status string `json:"status"`

	// StartedAt is when the scan was started (UTC).
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the scan finished (UTC).
	// Zero value if scan is still running.
	CompletedAt time.Time `json:"completed_at,omitempty"`

	// Duration is the scan duration in seconds.
	// Only set when scan is completed.
	Duration int `json:"duration_seconds,omitempty"`

	// Aggregate statistics (for fast filtering without reading JSONL files)

	// HostCount is the number of live hosts discovered.
	HostCount int `json:"host_count"`

	// ServiceCount is the number of services detected.
	ServiceCount int `json:"service_count"`

	// VulnCount contains vulnerability counts by severity.
	VulnCount VulnCounts `json:"vuln_count"`

	// StorageLocation indicates where the scan data is stored.
	// OSS: Local directory path (relative to workspace root)
	// Enterprise: S3 bucket prefix (e.g., "orgs/{org-id}/scans/{scan-id}")
	StorageLocation string `json:"storage_location,omitempty"`

	// ErrorMessage contains error details if the scan failed.
	ErrorMessage string `json:"error_message,omitempty"`

	// CreatedAt is when the scan metadata was first created (UTC).
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the scan metadata was last updated (UTC).
	UpdatedAt time.Time `json:"updated_at"`
}

// VulnCounts contains vulnerability counts by severity level.
type VulnCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

// Total returns the total number of vulnerabilities.
func (v VulnCounts) Total() int {
	return v.Critical + v.High + v.Medium + v.Low + v.Info
}

// ScanFilter specifies criteria for filtering and sorting scans.
type ScanFilter struct {
	// Status filters by scan status (empty = all statuses).
	Status string

	// Target filters by target substring match (empty = all targets).
	Target string

	// Limit is the maximum number of results to return (0 = no limit).
	Limit int

	// Offset is the number of results to skip (for pagination).
	Offset int

	// SortBy specifies the field to sort by.
	// Valid values: "time" (default), "severity", "target", "status"
	SortBy string

	// SortOrder specifies sort direction.
	// Valid values: "desc" (default), "asc"
	SortOrder string
}

// ScanUpdates specifies fields to update in a scan.
//
// Only non-zero fields are applied (partial update).
// Use pointers for optional fields to distinguish zero value from "not set".
type ScanUpdates struct {
	Status          *string     `json:"status,omitempty"`
	CompletedAt     *time.Time  `json:"completed_at,omitempty"`
	Duration        *int        `json:"duration_seconds,omitempty"`
	HostCount       *int        `json:"host_count,omitempty"`
	ServiceCount    *int        `json:"service_count,omitempty"`
	VulnCount       *VulnCounts `json:"vuln_count,omitempty"`
	ErrorMessage    *string     `json:"error_message,omitempty"`
	StorageLocation *string     `json:"storage_location,omitempty"`
}

// DataType represents the type of scan data file.
type DataType string

// Data file types (JSONL format).
const (
	// DataTypeMetadata is the scan metadata file (metadata.json).
	DataTypeMetadata DataType = "metadata.json"

	// DataTypeHosts is the discovered hosts file (hosts.jsonl).
	// Format: One JSON object per line, each representing a live host.
	DataTypeHosts DataType = "hosts.jsonl"

	// DataTypeServices is the detected services file (services.jsonl).
	// Format: One JSON object per line, each representing a service/port.
	DataTypeServices DataType = "services.jsonl"

	// DataTypeVulnerabilities is the vulnerabilities file (vulnerabilities.jsonl).
	// Format: One JSON object per line, each representing a vulnerability.
	DataTypeVulnerabilities DataType = "vulnerabilities.jsonl"

	// DataTypeBanners is the service banners file (banners.txt).
	// Format: Raw text, one banner per line.
	DataTypeBanners DataType = "banners.txt"
)

// String returns the string representation of DataType.
func (d DataType) String() string {
	return string(d)
}

// IsValid checks if the DataType is valid.
func (d DataType) IsValid() bool {
	switch d {
	case DataTypeMetadata, DataTypeHosts, DataTypeServices,
		DataTypeVulnerabilities, DataTypeBanners:
		return true
	default:
		return false
	}
}

// ScanStatus represents valid scan status values.
type ScanStatus string

// Valid scan statuses.
const (
	StatusPending   ScanStatus = "pending"
	StatusRunning   ScanStatus = "running"
	StatusCompleted ScanStatus = "completed"
	StatusFailed    ScanStatus = "failed"
	StatusCancelled ScanStatus = "canceled"
)

// String returns the string representation of ScanStatus.
func (s ScanStatus) String() string {
	return string(s)
}

// IsValid checks if the ScanStatus is valid.
func (s ScanStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status indicates the scan is finished.
func (s ScanStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCancelled
}
