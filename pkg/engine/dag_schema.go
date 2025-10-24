package engine

import (
	"fmt"
	"sort"
	"strings"
)

// DAGSchema represents a complete DAG definition loaded from YAML/JSON.
//
// This structure provides a declarative way to define scan workflows without
// writing Go code. It supports:
//   - Explicit dependency declarations (depends_on)
//   - Data flow contracts (consumes/produces)
//   - Per-node configuration
//   - Metadata and versioning
//
// Example:
//
//	dag := &DAGSchema{
//	    Name: "Port Scan Workflow",
//	    Version: "1.0",
//	    Nodes: []DAGNode{
//	        {
//	            ID: "discover",
//	            Module: "discovery.icmp-ping",
//	            Produces: []string{"discovery.live_hosts"},
//	        },
//	        {
//	            ID: "scan-ports",
//	            Module: "scan.tcp-port-scanner",
//	            DependsOn: []string{"discover"},
//	            Consumes: []string{"discovery.live_hosts"},
//	            Produces: []string{"scan.open_ports"},
//	        },
//	    },
//	}
type DAGSchema struct {
	// Name is a human-readable name for this DAG.
	Name string `yaml:"name" json:"name"`

	// Version is the DAG schema version (e.g., "1.0", "2.1").
	// Used for backward compatibility when schema changes.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Description provides additional context about this DAG's purpose.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Nodes are the execution units in this DAG.
	// Each node represents a module instance with its dependencies and data flow.
	Nodes []DAGNode `yaml:"nodes" json:"nodes"`

	// Metadata contains optional user-defined metadata.
	// Common fields: author, created, tags, etc.
	Metadata map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// DAGNode represents a single node in the DAG (a module instance).
//
// Each node:
//   - Has a unique ID within the DAG
//   - References a registered module by name
//   - Declares explicit dependencies via depends_on
//   - Specifies data flow contracts (consumes/produces)
//   - Can override module configuration
type DAGNode struct {
	// ID is a unique identifier for this node within the DAG.
	// Must be unique across all nodes in the same DAG.
	// Used in depends_on references.
	ID string `yaml:"id" json:"id"`

	// Module is the registered module name (e.g., "discovery.icmp-ping").
	// Must exist in the module registry.
	Module string `yaml:"module" json:"module"`

	// DependsOn lists node IDs that must complete before this node runs.
	// Forms the dependency graph edges.
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`

	// Consumes lists data keys this node reads from DataContext.
	// These keys must be produced by dependencies or available in initial context.
	Consumes []string `yaml:"consumes,omitempty" json:"consumes,omitempty"`

	// Produces lists data keys this node writes to DataContext.
	// Dependent nodes can consume these keys.
	Produces []string `yaml:"produces,omitempty" json:"produces,omitempty"`

	// Config is node-specific configuration passed to the module.
	// Structure depends on the module's requirements.
	Config map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// Validate performs comprehensive validation on the DAG definition.
//
// Validation checks:
//   - No duplicate node IDs
//   - All dependencies exist
//   - No cycles in dependency graph
//   - Data flow integrity (consumed keys are produced)
//   - All referenced modules are registered
//
// Returns a ValidationResult with errors and warnings.
func (d *DAGSchema) Validate() *ValidationResult {
	result := &ValidationResult{
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Check for empty DAG
	if len(d.Nodes) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "empty_dag",
			Message: "DAG has no nodes",
		})
		return result
	}

	// Build node index
	nodeIndex := make(map[string]*DAGNode)
	for i := range d.Nodes {
		node := &d.Nodes[i]
		if node.ID == "" {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "missing_id",
				Message: "Node is missing required 'id' field",
				NodeID:  fmt.Sprintf("index-%d", i),
			})
			continue
		}

		// Check for duplicate IDs
		if _, exists := nodeIndex[node.ID]; exists {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "duplicate_id",
				Message: fmt.Sprintf("Duplicate node ID: %s", node.ID),
				NodeID:  node.ID,
				Fix:     "Each node must have a unique ID. Rename one of the nodes.",
			})
			continue
		}

		nodeIndex[node.ID] = node
	}

	// Validate each node
	for i := range d.Nodes {
		node := &d.Nodes[i]
		d.validateNode(node, nodeIndex, result)
	}

	// Check for cycles (only if no critical errors)
	if len(result.Errors) == 0 {
		d.detectCycles(nodeIndex, result)
	}

	// Validate data flow (only if no critical errors)
	if len(result.Errors) == 0 {
		d.validateDataFlow(nodeIndex, result)
	}

	return result
}

// validateNode validates a single node's configuration.
func (d *DAGSchema) validateNode(node *DAGNode, nodeIndex map[string]*DAGNode, result *ValidationResult) {
	// Check module name
	if node.Module == "" {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "missing_module",
			Message: "Node is missing required 'module' field",
			NodeID:  node.ID,
			Fix:     "Specify the module name (e.g., 'discovery.icmp-ping')",
		})
	}

	// Validate dependencies exist
	for _, depID := range node.DependsOn {
		if _, exists := nodeIndex[depID]; !exists {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "missing_dependency",
				Message: fmt.Sprintf("Dependency '%s' does not exist", depID),
				NodeID:  node.ID,
				Fix:     fmt.Sprintf("Add a node with ID '%s' or remove from depends_on list", depID),
			})
		}
	}

	// Warn about self-dependency
	for _, depID := range node.DependsOn {
		if depID == node.ID {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Type:    "self_dependency",
				Message: "Node depends on itself (will be ignored)",
				NodeID:  node.ID,
			})
		}
	}

	// Warn if node produces nothing (possible sink node)
	if len(node.Produces) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "no_output",
			Message: "Node produces no output (sink node)",
			NodeID:  node.ID,
		})
	}
}

// detectCycles detects circular dependencies using DFS-based cycle detection.
func (d *DAGSchema) detectCycles(nodeIndex map[string]*DAGNode, result *ValidationResult) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true
		path = append(path, nodeID)

		node := nodeIndex[nodeID]
		for _, depID := range node.DependsOn {
			// Skip self-dependencies (already warned)
			if depID == nodeID {
				continue
			}

			if !visited[depID] {
				if dfs(depID) {
					return true
				}
			} else if recStack[depID] {
				// Found cycle - build cycle path
				cycleStart := -1
				for i, id := range path {
					if id == depID {
						cycleStart = i
						break
					}
				}
				cyclePath := append(path[cycleStart:], depID)

				result.Errors = append(result.Errors, ValidationError{
					Type:    "cycle",
					Message: fmt.Sprintf("Circular dependency detected: %s", strings.Join(cyclePath, " -> ")),
					NodeID:  nodeID,
					Fix:     "Remove one of the dependencies to break the cycle",
				})
				return true
			}
		}

		recStack[nodeID] = false
		path = path[:len(path)-1]
		return false
	}

	// Check all nodes (graph might not be connected)
	for nodeID := range nodeIndex {
		if !visited[nodeID] {
			if dfs(nodeID) {
				return // Stop after first cycle
			}
		}
	}
}

// validateDataFlow ensures all consumed keys are produced by dependencies.
func (d *DAGSchema) validateDataFlow(nodeIndex map[string]*DAGNode, result *ValidationResult) {
	// Build map of what each node produces
	producers := make(map[string][]string) // key -> node IDs that produce it
	for nodeID, node := range nodeIndex {
		for _, key := range node.Produces {
			producers[key] = append(producers[key], nodeID)
		}
	}

	// Check each node's consumed keys
	for _, node := range nodeIndex {
		for _, consumedKey := range node.Consumes {
			// Check if any dependency produces this key
			producedByDep := false

			// Get all dependencies (direct and transitive)
			allDeps := d.getTransitiveDependencies(node.ID, nodeIndex)

			for _, depID := range allDeps {
				dep := nodeIndex[depID]
				for _, producedKey := range dep.Produces {
					if producedKey == consumedKey {
						producedByDep = true
						break
					}
				}
				if producedByDep {
					break
				}
			}

			// Special case: keys starting with "config." are external inputs
			if strings.HasPrefix(consumedKey, "config.") {
				continue
			}

			if !producedByDep {
				// Check if it's produced at all
				if producingNodes, exists := producers[consumedKey]; exists {
					result.Warnings = append(result.Warnings, ValidationWarning{
						Type:    "data_flow",
						Message: fmt.Sprintf("Node consumes '%s' but it's produced by non-dependency nodes: %s", consumedKey, strings.Join(producingNodes, ", ")),
						NodeID:  node.ID,
					})
				} else {
					result.Errors = append(result.Errors, ValidationError{
						Type:    "data_flow",
						Message: fmt.Sprintf("Node consumes '%s' but no dependency produces it", consumedKey),
						NodeID:  node.ID,
						Fix:     fmt.Sprintf("Add a dependency that produces '%s' or remove from consumes list", consumedKey),
					})
				}
			}
		}
	}
}

// getTransitiveDependencies returns all direct and transitive dependencies of a node.
func (d *DAGSchema) getTransitiveDependencies(nodeID string, nodeIndex map[string]*DAGNode) []string {
	visited := make(map[string]bool)
	result := make([]string, 0)

	var dfs func(id string)
	dfs = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true

		node := nodeIndex[id]
		for _, depID := range node.DependsOn {
			if depID == id {
				continue // Skip self-dependencies
			}
			result = append(result, depID)
			dfs(depID)
		}
	}

	dfs(nodeID)
	return result
}

// GetExecutionOrder returns nodes in topological order (dependencies first).
//
// Returns error if DAG has cycles (should validate first).
func (d *DAGSchema) GetExecutionOrder() ([]string, error) {
	nodeIndex := make(map[string]*DAGNode)
	for i := range d.Nodes {
		nodeIndex[d.Nodes[i].ID] = &d.Nodes[i]
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	result := make([]string, 0, len(d.Nodes))

	var dfs func(nodeID string) error
	dfs = func(nodeID string) error {
		if recStack[nodeID] {
			return fmt.Errorf("cycle detected involving node: %s", nodeID)
		}
		if visited[nodeID] {
			return nil
		}

		visited[nodeID] = true
		recStack[nodeID] = true

		node := nodeIndex[nodeID]
		for _, depID := range node.DependsOn {
			if depID == nodeID {
				continue // Skip self-dependencies
			}
			if err := dfs(depID); err != nil {
				return err
			}
		}

		recStack[nodeID] = false
		result = append(result, nodeID)
		return nil
	}

	// Process all nodes (graph might not be connected)
	nodeIDs := make([]string, 0, len(nodeIndex))
	for id := range nodeIndex {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs) // Deterministic ordering

	for _, nodeID := range nodeIDs {
		if !visited[nodeID] {
			if err := dfs(nodeID); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// ValidationResult contains the results of DAG validation.
type ValidationResult struct {
	// Errors are critical issues that prevent DAG execution.
	Errors []ValidationError

	// Warnings are potential issues that don't prevent execution.
	Warnings []ValidationWarning
}

// IsValid returns true if there are no errors.
func (r *ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// String returns a human-readable validation summary.
func (r *ValidationResult) String() string {
	if r.IsValid() && len(r.Warnings) == 0 {
		return "✅ DAG is valid"
	}

	var sb strings.Builder

	if len(r.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("Found %d validation error(s):\n", len(r.Errors)))
		for i, err := range r.Errors {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, err.Type, err.Message))
			if err.NodeID != "" {
				sb.WriteString(fmt.Sprintf("     Node: %s\n", err.NodeID))
			}
			if err.Fix != "" {
				sb.WriteString(fmt.Sprintf("     Fix: %s\n", err.Fix))
			}
		}
	}

	if len(r.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("\nFound %d warning(s):\n", len(r.Warnings)))
		for i, warn := range r.Warnings {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, warn.Type, warn.Message))
			if warn.NodeID != "" {
				sb.WriteString(fmt.Sprintf("     Node: %s\n", warn.NodeID))
			}
		}
	}

	return sb.String()
}

// ValidationError represents a critical validation error.
type ValidationError struct {
	// Type categorizes the error (e.g., "cycle", "missing_dependency").
	Type string

	// Message is a human-readable error description.
	Message string

	// NodeID identifies the node where the error occurred (if applicable).
	NodeID string

	// Fix suggests how to resolve the error.
	Fix string
}

// ValidationWarning represents a non-critical validation warning.
type ValidationWarning struct {
	// Type categorizes the warning.
	Type string

	// Message is a human-readable warning description.
	Message string

	// NodeID identifies the node where the warning occurred (if applicable).
	NodeID string
}

// ToDAGDefinition converts a DAGSchema to a DAGDefinition for orchestrator execution.
//
// This method:
//   - Validates the DAG schema first
//   - Converts DAGNode → DAGNodeConfig
//   - Stores explicit dependencies in node config for orchestrator
//
// Returns error if validation fails.
func (d *DAGSchema) ToDAGDefinition() (*DAGDefinition, error) {
	// Validate schema first
	validationResult := d.Validate()
	if !validationResult.IsValid() {
		return nil, fmt.Errorf("DAG validation failed: %s", validationResult.String())
	}

	// Convert nodes
	nodes := make([]DAGNodeConfig, 0, len(d.Nodes))
	for _, node := range d.Nodes {
		// Make a copy of the config to avoid modifying the original
		var config map[string]interface{}
		if node.Config != nil {
			config = make(map[string]interface{}, len(node.Config))
			for k, v := range node.Config {
				config[k] = v
			}
		} else {
			config = make(map[string]interface{})
		}

		nodeConfig := DAGNodeConfig{
			InstanceID: node.ID,
			ModuleType: node.Module,
			Config:     config,
		}

		// Store explicit dependencies in config for orchestrator to use
		// Convert []string to []interface{} for consistent type handling
		if len(node.DependsOn) > 0 {
			depends := make([]interface{}, len(node.DependsOn))
			for i, dep := range node.DependsOn {
				depends[i] = dep
			}
			nodeConfig.Config["__depends_on"] = depends
		}

		// Store explicit consumes/produces for validation
		if len(node.Consumes) > 0 {
			consumes := make([]interface{}, len(node.Consumes))
			for i, c := range node.Consumes {
				consumes[i] = c
			}
			nodeConfig.Config["__consumes"] = consumes
		}
		if len(node.Produces) > 0 {
			produces := make([]interface{}, len(node.Produces))
			for i, p := range node.Produces {
				produces[i] = p
			}
			nodeConfig.Config["__produces"] = produces
		}

		nodes = append(nodes, nodeConfig)
	}

	return &DAGDefinition{
		Name:        d.Name,
		Description: d.Description,
		Nodes:       nodes,
	}, nil
}
