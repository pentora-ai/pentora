package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// Ensure that when config.targets comes with wrong type, we fall back to legacy SetInitial
func TestOrchestrator_InitialInputs_ConfigTargets_WrongType_Fallback(t *testing.T) {
	// Minimal DAG with a no-op module to satisfy orchestrator requirements
	dag := &DAGDefinition{Name: "seed-fallback", Nodes: []DAGNodeConfig{{InstanceID: "noop1", ModuleType: "noop"}}}
	RegisterModuleFactory("noop", func() Module { return &minimalTestModule{meta: ModuleMetadata{Produces: nil}} })
	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	// Pass wrong type (string instead of []string)
	_, runErr := orc.Run(context.Background(), map[string]interface{}{"config.targets": "10.0.0.1"})
	require.NoError(t, runErr)

	// Legacy storage should keep raw value accessible via GetAll
	all := orc.dataCtx.GetAll()
	v, ok := all["config.targets"]
	require.True(t, ok)
	require.Equal(t, "10.0.0.1", v)
}
