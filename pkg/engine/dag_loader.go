package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadDAGFromFile loads a DAG definition from a YAML or JSON file.
//
// The file format is determined by extension:
//   - .yaml, .yml → YAML
//   - .json → JSON
//
// Returns error if:
//   - File doesn't exist or can't be read
//   - File format is invalid
//   - Validation fails (unless skipValidation is true)
//
// Example:
//
//	dag, err := LoadDAGFromFile("scans/port-scan.yaml", false)
//	if err != nil {
//	    return err
//	}
func LoadDAGFromFile(path string, skipValidation bool) (*DAGSchema, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(path))
	var dag DAGSchema

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &dag); err != nil {
			return nil, fmt.Errorf("parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &dag); err != nil {
			return nil, fmt.Errorf("parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s (use .yaml, .yml, or .json)", ext)
	}

	// Validate unless skipped
	if !skipValidation {
		result := dag.Validate()
		if !result.IsValid() {
			return nil, fmt.Errorf("validation failed:\n%s", result.String())
		}

		// Log warnings if present
		if len(result.Warnings) > 0 {
			// Warnings don't prevent loading, but should be visible
			// TODO: Use structured logging when available
			fmt.Fprintf(os.Stderr, "DAG loaded with warnings:\n%s\n", result.String())
		}
	}

	return &dag, nil
}

// LoadDAGFromBytes loads a DAG definition from raw YAML or JSON bytes.
//
// The format is auto-detected by attempting YAML first, then JSON.
//
// Returns error if:
//   - Data is invalid YAML/JSON
//   - Validation fails (unless skipValidation is true)
func LoadDAGFromBytes(data []byte, skipValidation bool) (*DAGSchema, error) {
	var dag DAGSchema

	// Try YAML first (more permissive, handles JSON too)
	if err := yaml.Unmarshal(data, &dag); err != nil {
		// Try JSON
		if jsonErr := json.Unmarshal(data, &dag); jsonErr != nil {
			return nil, fmt.Errorf("parse YAML/JSON: yaml=%v, json=%v", err, jsonErr)
		}
	}

	// Validate unless skipped
	if !skipValidation {
		result := dag.Validate()
		if !result.IsValid() {
			return nil, fmt.Errorf("validation failed:\n%s", result.String())
		}

		if len(result.Warnings) > 0 {
			fmt.Fprintf(os.Stderr, "DAG loaded with warnings:\n%s\n", result.String())
		}
	}

	return &dag, nil
}

// SaveDAGToFile saves a DAG definition to a YAML or JSON file.
//
// The file format is determined by extension:
//   - .yaml, .yml → YAML
//   - .json → JSON
//
// YAML output is formatted with 2-space indentation.
// JSON output is formatted with 2-space indentation.
//
// Example:
//
//	dag := &DAGSchema{
//	    Name: "My Scan",
//	    Nodes: []DAGNode{...},
//	}
//	err := SaveDAGToFile(dag, "scans/my-scan.yaml")
func SaveDAGToFile(dag *DAGSchema, path string) error {
	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(path))

	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(dag)
		if err != nil {
			return fmt.Errorf("marshal YAML: %w", err)
		}
	case ".json":
		data, err = json.MarshalIndent(dag, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s (use .yaml, .yml, or .json)", ext)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// DAGFromModules constructs a DAG definition from a list of modules.
//
// This is useful for converting programmatic module configurations
// to declarative YAML/JSON definitions.
//
// The function:
//   - Generates unique node IDs from module names
//   - Extracts consumes/produces from module metadata
//   - Infers dependencies from data flow (if a module consumes what another produces)
//
// Note: This is a best-effort conversion. Manual adjustment may be needed
// for complex dependency patterns.
func DAGFromModules(name string, modules []Module) (*DAGSchema, error) {
	dag := &DAGSchema{
		Name:    name,
		Version: "1.0",
		Nodes:   make([]DAGNode, 0, len(modules)),
	}

	// Build map of what each module produces
	producers := make(map[string][]int) // key -> module indices

	for i, mod := range modules {
		meta := mod.Metadata()

		// Generate node ID from module name
		nodeID := generateNodeID(meta.Name, i)

		// Extract data keys from contracts
		consumes := extractDataKeys(meta.Consumes)
		produces := extractDataKeys(meta.Produces)

		// Track what this module produces
		for _, key := range produces {
			producers[key] = append(producers[key], i)
		}

		dag.Nodes = append(dag.Nodes, DAGNode{
			ID:       nodeID,
			Module:   meta.Name,
			Consumes: consumes,
			Produces: produces,
		})
	}

	// Infer dependencies from data flow
	for i := range dag.Nodes {
		node := &dag.Nodes[i]
		deps := make(map[string]bool) // Use map to deduplicate

		for _, consumedKey := range node.Consumes {
			// Find modules that produce this key
			if producerIndices, exists := producers[consumedKey]; exists {
				for _, j := range producerIndices {
					if j < i { // Only depend on earlier modules
						deps[dag.Nodes[j].ID] = true
					}
				}
			}
		}

		// Convert map to slice
		node.DependsOn = make([]string, 0, len(deps))
		for depID := range deps {
			node.DependsOn = append(node.DependsOn, depID)
		}
	}

	return dag, nil
}

// generateNodeID creates a unique node ID from a module name and index.
func generateNodeID(moduleName string, index int) string {
	// Convert "discovery.icmp-ping" → "discover-icmp"
	parts := strings.Split(moduleName, ".")
	if len(parts) > 1 {
		// Use last part and simplify
		name := parts[len(parts)-1]
		name = strings.TrimSuffix(name, "-scanner")
		name = strings.TrimSuffix(name, "-parser")
		name = strings.TrimSuffix(name, "-grabber")
		return fmt.Sprintf("%s-%d", name, index)
	}
	return fmt.Sprintf("node-%d", index)
}

// extractDataKeys extracts just the key names from DataContractEntry list.
func extractDataKeys(contracts []DataContractEntry) []string {
	keys := make([]string, len(contracts))
	for i, contract := range contracts {
		keys[i] = contract.Key
	}
	return keys
}
