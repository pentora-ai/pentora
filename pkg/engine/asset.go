// pkg/engine/asset.go
package engine

import (
	"time"
)

// FindingSeverity defines the severity of a finding.
type FindingSeverity string

const (
	SeverityCritical    FindingSeverity = "critical"
	SeverityHigh        FindingSeverity = "high"
	SeverityMedium      FindingSeverity = "medium"
	SeverityLow         FindingSeverity = "low"
	SeverityInfo        FindingSeverity = "informational"
	SeverityUndetermined FindingSeverity = "undetermined"
)

// VulnerabilityFinding details a specific vulnerability found.
type VulnerabilityFinding struct {
	ID               string   `json:"id" yaml:"id"` // CVE ID, Pentora Vuln ID, etc.
	SourceModule     string   `json:"source_module" yaml:"source_module"` // Which module instance found this
	Summary          string   `json:"summary" yaml:"summary"`
	Severity         FindingSeverity `json:"severity" yaml:"severity"`
	Description      string   `json:"description,omitempty" yaml:"description,omitempty"`
	References       []string `json:"references,omitempty" yaml:"references,omitempty"`
	Remediation      string   `json:"remediation,omitempty" yaml:"remediation,omitempty"`
	// Evidence string `json:"evidence,omitempty" yaml:"evidence,omitempty"` // Specific evidence
}

// ServiceDetails contains information about the service running on a port.
type ServiceDetails struct {
	Name             string                 `json:"name,omitempty" yaml:"name,omitempty"`             // e.g., SSH, HTTP
	Product          string                 `json:"product,omitempty" yaml:"product,omitempty"`       // e.g., OpenSSH, Nginx (parser'dan gelen daha spesifik ürün adı)
	Version          string                 `json:"version,omitempty" yaml:"version,omitempty"`       // e.g., 8.2p1, 1.22.1
	RawBanner        string                 `json:"raw_banner,omitempty" yaml:"raw_banner,omitempty"` // Raw banner captured
	IsTLS            bool                   `json:"is_tls,omitempty" yaml:"is_tls,omitempty"`
	ParsedAttributes map[string]interface{} `json:"parsed_attributes,omitempty" yaml:"parsed_attributes,omitempty"` // HTTP headers, SSH specific details, etc.
}

// PortProfile details information about a specific open port on a target.
type PortProfile struct {
	PortNumber      int                    `json:"port_number" yaml:"port_number"`
	Protocol        string                 `json:"protocol" yaml:"protocol"` // "tcp", "udp"
	Status          string                 `json:"status" yaml:"status"`     // "open", "filtered", "closed"
	Service         ServiceDetails         `json:"service,omitempty" yaml:"service,omitempty"`
	Vulnerabilities []VulnerabilityFinding `json:"vulnerabilities,omitempty" yaml:"vulnerabilities,omitempty"`
}

// AssetProfile represents a comprehensive profile for a single scanned target.
type AssetProfile struct {
	Target               string                 `json:"target" yaml:"target"` // Original target string (IP, hostname, CIDR)
	ResolvedIPs          map[string]time.Time   `json:"resolved_ips,omitempty" yaml:"resolved_ips,omitempty"` // Map of IP to first_seen_alive_time
	Hostnames            []string               `json:"hostnames,omitempty" yaml:"hostnames,omitempty"`
	IsAlive              bool                   `json:"is_alive" yaml:"is_alive"` // If any IP resolved from target is alive
	FirstSeenAlive       time.Time              `json:"first_seen_alive,omitempty" yaml:"first_seen_alive,omitempty"`
	LastObservationTime  time.Time              `json:"last_observation_time" yaml:"last_observation_time"` // When data for this asset was last updated
	OpenPorts            map[string][]PortProfile `json:"open_ports_by_ip,omitempty" yaml:"open_ports_by_ip,omitempty"` // Keyed by IP address
	TotalVulnerabilities int                    `json:"total_vulnerabilities" yaml:"total_vulnerabilities"`
	// OperatingSystem string `json:"operating_system,omitempty" yaml:"operating_system,omitempty"`
	// MACAddress string `json:"mac_address,omitempty" yaml:"mac_address,omitempty"`
	ErrorsEncountered []string `json:"errors_encountered,omitempty" yaml:"errors_encountered,omitempty"` // Errors specific to this asset during scan
}