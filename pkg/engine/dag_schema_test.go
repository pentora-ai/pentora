package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDAGSchema_Validate_EmptyDAG(t *testing.T) {
	dag := &DAGSchema{
		Name:  "Empty DAG",
		Nodes: []DAGNode{},
	}

	result := dag.Validate()
	require.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	require.Equal(t, "empty_dag", result.Errors[0].Type)
}

func TestDAGSchema_Validate_MissingNodeID(t *testing.T) {
	dag := &DAGSchema{
		Name: "Missing ID",
		Nodes: []DAGNode{
			{
				Module:   "test.module",
				Produces: []string{"data.output"},
			},
		},
	}

	result := dag.Validate()
	require.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	require.Equal(t, "missing_id", result.Errors[0].Type)
}

func TestDAGSchema_Validate_DuplicateNodeID(t *testing.T) {
	dag := &DAGSchema{
		Name: "Duplicate IDs",
		Nodes: []DAGNode{
			{
				ID:       "scanner",
				Module:   "scan.tcp",
				Produces: []string{"scan.tcp_ports"},
			},
			{
				ID:       "scanner", // Duplicate!
				Module:   "scan.udp",
				Produces: []string{"scan.udp_ports"},
			},
		},
	}

	result := dag.Validate()
	require.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	require.Equal(t, "duplicate_id", result.Errors[0].Type)
	require.Contains(t, result.Errors[0].Message, "scanner")
}

func TestDAGSchema_Validate_MissingModule(t *testing.T) {
	dag := &DAGSchema{
		Name: "Missing Module",
		Nodes: []DAGNode{
			{
				ID:       "node-1",
				Produces: []string{"data.output"},
			},
		},
	}

	result := dag.Validate()
	require.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	require.Equal(t, "missing_module", result.Errors[0].Type)
}

func TestDAGSchema_Validate_MissingDependency(t *testing.T) {
	dag := &DAGSchema{
		Name: "Missing Dependency",
		Nodes: []DAGNode{
			{
				ID:        "scanner",
				Module:    "scan.tcp",
				DependsOn: []string{"discoverer"}, // This node doesn't exist!
				Produces:  []string{"scan.ports"},
			},
		},
	}

	result := dag.Validate()
	require.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	require.Equal(t, "missing_dependency", result.Errors[0].Type)
	require.Contains(t, result.Errors[0].Message, "discoverer")
}

func TestDAGSchema_Validate_SimpleCycle(t *testing.T) {
	dag := &DAGSchema{
		Name: "Simple Cycle",
		Nodes: []DAGNode{
			{
				ID:        "node-a",
				Module:    "test.module",
				DependsOn: []string{"node-b"},
				Produces:  []string{"data.a"},
			},
			{
				ID:        "node-b",
				Module:    "test.module",
				DependsOn: []string{"node-a"}, // Cycle!
				Produces:  []string{"data.b"},
			},
		},
	}

	result := dag.Validate()
	require.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	require.Equal(t, "cycle", result.Errors[0].Type)
	require.Contains(t, result.Errors[0].Message, "node-a")
	require.Contains(t, result.Errors[0].Message, "node-b")
}

func TestDAGSchema_Validate_ComplexCycle(t *testing.T) {
	dag := &DAGSchema{
		Name: "Complex Cycle",
		Nodes: []DAGNode{
			{
				ID:        "node-a",
				Module:    "test.module",
				DependsOn: []string{"node-b"},
				Produces:  []string{"data.a"},
			},
			{
				ID:        "node-b",
				Module:    "test.module",
				DependsOn: []string{"node-c"},
				Produces:  []string{"data.b"},
			},
			{
				ID:        "node-c",
				Module:    "test.module",
				DependsOn: []string{"node-a"}, // Cycle: a->b->c->a
				Produces:  []string{"data.c"},
			},
		},
	}

	result := dag.Validate()
	require.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	require.Equal(t, "cycle", result.Errors[0].Type)
}

func TestDAGSchema_Validate_SelfDependency(t *testing.T) {
	dag := &DAGSchema{
		Name: "Self Dependency",
		Nodes: []DAGNode{
			{
				ID:        "node-a",
				Module:    "test.module",
				DependsOn: []string{"node-a"}, // Self-dependency
				Produces:  []string{"data.a"},
			},
		},
	}

	result := dag.Validate()
	require.True(t, result.IsValid()) // Self-dependency is a warning, not error
	require.Len(t, result.Warnings, 1)
	require.Equal(t, "self_dependency", result.Warnings[0].Type)
}

func TestDAGSchema_Validate_DataFlowError(t *testing.T) {
	dag := &DAGSchema{
		Name: "Data Flow Error",
		Nodes: []DAGNode{
			{
				ID:       "scanner",
				Module:   "scan.tcp",
				Consumes: []string{"discovery.live_hosts"}, // Never produced!
				Produces: []string{"scan.ports"},
			},
		},
	}

	result := dag.Validate()
	require.False(t, result.IsValid())
	require.Len(t, result.Errors, 1)
	require.Equal(t, "data_flow", result.Errors[0].Type)
	require.Contains(t, result.Errors[0].Message, "discovery.live_hosts")
}

func TestDAGSchema_Validate_DataFlowValid(t *testing.T) {
	dag := &DAGSchema{
		Name: "Valid Data Flow",
		Nodes: []DAGNode{
			{
				ID:       "discover",
				Module:   "discovery.icmp",
				Produces: []string{"discovery.live_hosts"},
			},
			{
				ID:        "scanner",
				Module:    "scan.tcp",
				DependsOn: []string{"discover"},
				Consumes:  []string{"discovery.live_hosts"},
				Produces:  []string{"scan.ports"},
			},
		},
	}

	result := dag.Validate()
	require.True(t, result.IsValid())
	require.Empty(t, result.Errors)
}

func TestDAGSchema_Validate_ConfigKeys(t *testing.T) {
	// Keys starting with "config." are external inputs and don't need producers
	dag := &DAGSchema{
		Name: "Config Keys",
		Nodes: []DAGNode{
			{
				ID:       "scanner",
				Module:   "scan.tcp",
				Consumes: []string{"config.targets", "config.ports"}, // External inputs
				Produces: []string{"scan.ports"},
			},
		},
	}

	result := dag.Validate()
	require.True(t, result.IsValid())
	require.Empty(t, result.Errors)
}

func TestDAGSchema_Validate_NoOutputWarning(t *testing.T) {
	dag := &DAGSchema{
		Name: "No Output Warning",
		Nodes: []DAGNode{
			{
				ID:       "discover",
				Module:   "discovery.icmp",
				Produces: []string{"discovery.live_hosts"},
			},
			{
				ID:        "sink",
				Module:    "reporting.console",
				DependsOn: []string{"discover"},
				Consumes:  []string{"discovery.live_hosts"},
				// No produces - sink node
			},
		},
	}

	result := dag.Validate()
	require.True(t, result.IsValid())
	require.Len(t, result.Warnings, 1)
	require.Equal(t, "no_output", result.Warnings[0].Type)
	require.Equal(t, "sink", result.Warnings[0].NodeID)
}

func TestDAGSchema_Validate_DataFlowWarning(t *testing.T) {
	// Node consumes data produced by a non-dependency
	dag := &DAGSchema{
		Name: "Data Flow Warning",
		Nodes: []DAGNode{
			{
				ID:       "producer",
				Module:   "test.module",
				Produces: []string{"data.x"},
			},
			{
				ID:       "consumer",
				Module:   "test.module",
				Consumes: []string{"data.x"}, // Produced by 'producer' but not a dependency
				Produces: []string{"data.y"},
			},
		},
	}

	result := dag.Validate()
	require.True(t, result.IsValid()) // Warning, not error
	require.Len(t, result.Warnings, 1)
	require.Equal(t, "data_flow", result.Warnings[0].Type)
	require.Contains(t, result.Warnings[0].Message, "producer")
}

func TestDAGSchema_GetExecutionOrder_Simple(t *testing.T) {
	dag := &DAGSchema{
		Name: "Simple Sequential",
		Nodes: []DAGNode{
			{
				ID:       "node-1",
				Module:   "test.module",
				Produces: []string{"data.1"},
			},
			{
				ID:        "node-2",
				Module:    "test.module",
				DependsOn: []string{"node-1"},
				Consumes:  []string{"data.1"},
				Produces:  []string{"data.2"},
			},
			{
				ID:        "node-3",
				Module:    "test.module",
				DependsOn: []string{"node-2"},
				Consumes:  []string{"data.2"},
				Produces:  []string{"data.3"},
			},
		},
	}

	order, err := dag.GetExecutionOrder()
	require.NoError(t, err)
	require.Len(t, order, 3)

	// node-1 must come before node-2
	idx1 := indexOf(order, "node-1")
	idx2 := indexOf(order, "node-2")
	idx3 := indexOf(order, "node-3")
	require.True(t, idx1 < idx2)
	require.True(t, idx2 < idx3)
}

func TestDAGSchema_GetExecutionOrder_Diamond(t *testing.T) {
	dag := &DAGSchema{
		Name: "Diamond Pattern",
		Nodes: []DAGNode{
			{
				ID:       "root",
				Module:   "test.module",
				Produces: []string{"data.root"},
			},
			{
				ID:        "left",
				Module:    "test.module",
				DependsOn: []string{"root"},
				Consumes:  []string{"data.root"},
				Produces:  []string{"data.left"},
			},
			{
				ID:        "right",
				Module:    "test.module",
				DependsOn: []string{"root"},
				Consumes:  []string{"data.root"},
				Produces:  []string{"data.right"},
			},
			{
				ID:        "merge",
				Module:    "test.module",
				DependsOn: []string{"left", "right"},
				Consumes:  []string{"data.left", "data.right"},
				Produces:  []string{"data.merged"},
			},
		},
	}

	order, err := dag.GetExecutionOrder()
	require.NoError(t, err)
	require.Len(t, order, 4)

	// root must come first
	require.Equal(t, "root", order[0])

	// left and right can be in any order (parallel)
	idxLeft := indexOf(order, "left")
	idxRight := indexOf(order, "right")
	idxMerge := indexOf(order, "merge")

	// But merge must come after both
	require.True(t, idxLeft < idxMerge)
	require.True(t, idxRight < idxMerge)
}

func TestDAGSchema_GetExecutionOrder_Cycle(t *testing.T) {
	dag := &DAGSchema{
		Name: "Cycle",
		Nodes: []DAGNode{
			{
				ID:        "node-a",
				Module:    "test.module",
				DependsOn: []string{"node-b"},
			},
			{
				ID:        "node-b",
				Module:    "test.module",
				DependsOn: []string{"node-a"},
			},
		},
	}

	_, err := dag.GetExecutionOrder()
	require.Error(t, err)
	require.Contains(t, err.Error(), "cycle")
}

func TestValidationResult_String_Valid(t *testing.T) {
	result := &ValidationResult{
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
	}

	str := result.String()
	require.Contains(t, str, "✅ DAG is valid")
}

func TestValidationResult_String_WithErrors(t *testing.T) {
	result := &ValidationResult{
		Errors: []ValidationError{
			{
				Type:    "cycle",
				Message: "Circular dependency detected: a -> b -> a",
				NodeID:  "a",
				Fix:     "Remove one of the dependencies",
			},
		},
		Warnings: []ValidationWarning{},
	}

	str := result.String()
	require.Contains(t, str, "Found 1 validation error(s)")
	require.Contains(t, str, "[cycle]")
	require.Contains(t, str, "Circular dependency")
	require.Contains(t, str, "Node: a")
	require.Contains(t, str, "Fix:")
}

func TestValidationResult_String_WithWarnings(t *testing.T) {
	result := &ValidationResult{
		Errors: []ValidationError{},
		Warnings: []ValidationWarning{
			{
				Type:    "no_output",
				Message: "Node produces no output",
				NodeID:  "sink",
			},
		},
	}

	str := result.String()
	require.Contains(t, str, "Found 1 warning(s)")
	require.Contains(t, str, "[no_output]")
	require.Contains(t, str, "Node: sink")
}

func TestDAGSchema_GetTransitiveDependencies(t *testing.T) {
	dag := &DAGSchema{
		Name: "Transitive Dependencies",
		Nodes: []DAGNode{
			{
				ID:       "node-a",
				Module:   "test.module",
				Produces: []string{"data.a"},
			},
			{
				ID:        "node-b",
				Module:    "test.module",
				DependsOn: []string{"node-a"},
				Produces:  []string{"data.b"},
			},
			{
				ID:        "node-c",
				Module:    "test.module",
				DependsOn: []string{"node-b"},
				Produces:  []string{"data.c"},
			},
		},
	}

	nodeIndex := make(map[string]*DAGNode)
	for i := range dag.Nodes {
		nodeIndex[dag.Nodes[i].ID] = &dag.Nodes[i]
	}

	// node-c transitively depends on both node-b and node-a
	deps := dag.getTransitiveDependencies("node-c", nodeIndex)
	require.Contains(t, deps, "node-b")
	require.Contains(t, deps, "node-a")
}

// Helper function to find index of element in slice
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
