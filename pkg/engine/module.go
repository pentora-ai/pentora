// pkg/engine/module.go
package engine

import (
	"context"
	"time"
)

// ModuleType represents the category of the module.
type ModuleType string

const (
	DiscoveryModuleType     ModuleType = "discovery"     // For host, port, or service discovery
	ScanModuleType          ModuleType = "scan"          // For active scanning, probing, banner grabbing
	ParseModuleType         ModuleType = "parse"         // For parsing raw data into structured information
	EvaluationModuleType    ModuleType = "evaluation"    // For vulnerability checks, compliance checks, etc.
	ReportingModuleType     ModuleType = "reporting"     // For generating reports
	OutputModuleType        ModuleType = "output"        // For sending results to different sinks
	OrchestrationModuleType ModuleType = "orchestration" // Meta-modules that can manage other modules
)

// ModuleMetadata holds common information for all modules.
type ModuleMetadata struct {
	ID          string     // Unique identifier for the module instance in a DAG
	Name        string     // Human-readable name of the module type (e.g., "ICMP Ping Discovery")
	Version     string     // Version of the module implementation
	Description string     // Brief description of what the module does
	Type        ModuleType // Category of the module (discovery, scan, etc.)
	Author      string     // Author of the module
	Tags        []string   // Tags for categorization or filtering

	// Defines what data keys this module can produce.
	// Example: ["discovery.live_hosts", "asset.ip_addresses"]
	Produces []string

	// Defines what data keys this module consumes from the data context or previous modules.
	// Example: ["config.targets", "discovery.live_hosts"]
	Consumes []string

	// Defines module-specific configuration parameters and their types/defaults.
	// This could be a more structured type or a map for flexibility.
	ConfigSchema map[string]ParameterDefinition
}

// ParameterDefinition describes a configuration parameter for a module.
type ParameterDefinition struct {
	Description string
	Type        string // e.g., "string", "int", "bool", "duration", "[]string"
	Required    bool
	Default     interface{}
}

// ModuleOutput represents the data produced by a module's execution.
type ModuleOutput struct {
	// FromModuleName is the ID of the module instance that produced this output.
	FromModuleName string
	// DataKey is a string key identifying the type or nature of the data.
	// Allows consumers to understand what this data represents.
	// e.g., "discovery.live_hosts", "service.banner.ssh", "vulnerability.CVE-2021-44228"
	DataKey string
	// Data is the actual payload.
	Data interface{}
	// Error if the module execution failed for this specific output.
	Error error
	// Timestamp when the data was produced.
	Timestamp time.Time
	// Target associated with this output, if applicable (e.g., IP address, hostname).
	Target string
}

// Module is the core interface that all functional units in Pentora should implement.
type Module interface {
	// Metadata returns descriptive information about the module.
	Metadata() ModuleMetadata

	// Init initializes the module with its specific configuration.
	// The config map is typically derived from the DAG definition.
	Init(moduleConfig map[string]interface{}) error

	// Execute runs the module's main logic.
	// It takes the current execution context, a map of input data (keyed by DataKey),
	// and a channel to send its outputs.
	Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- ModuleOutput) error
}
