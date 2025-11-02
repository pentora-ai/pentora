package engine

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// minimalTestModule emits a single output with provided key/data.
type minimalTestModule struct {
	meta    ModuleMetadata
	outKey  string
	outData interface{}
}

func (m *minimalTestModule) Metadata() ModuleMetadata { return m.meta }
func (m *minimalTestModule) Init(instanceID string, moduleConfig map[string]interface{}) error {
	return nil
}

func (m *minimalTestModule) Execute(ctx context.Context, inputs map[string]interface{}, ch chan<- ModuleOutput) error {
	ch <- ModuleOutput{DataKey: m.outKey, Data: m.outData}
	return nil
}

func TestOrchestrator_TypedWrite_ListAppendWhenSchemaRegistered(t *testing.T) {
	dag := &DAGDefinition{Name: "typed-list", Nodes: []DAGNodeConfig{{InstanceID: "n1", ModuleType: "test-min"}}}

	// Register factory for test module
	RegisterModuleFactory("test-min", func() Module {
		return &minimalTestModule{
			meta: ModuleMetadata{
				Name:        "test-min",
				Description: "emits one output",
				Consumes:    nil,
				Produces:    []DataContractEntry{{Key: "service.banner.tcp", Cardinality: CardinalityList}},
			},
			outKey: "service.banner.tcp",
			// We'll append a string as banner for test purposes
			outData: "ssh-banner",
		}
	})

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	// Pre-register schema for list key: []string
	err = orc.dataCtx.RegisterType("service.banner.tcp", reflect.TypeOf([]string{}), CardinalityList)
	require.NoError(t, err)

	// Provide initial inputs to exercise typed seeding path as well
	_, runErr := orc.Run(context.Background(), map[string]interface{}{"config.targets": []string{"10.0.0.1"}})
	require.NoError(t, runErr)

	v, err := orc.dataCtx.GetValue("service.banner.tcp")
	require.NoError(t, err)
	// Should be []string with one element "ssh-banner"
	slice, ok := v.([]string)
	require.True(t, ok)
	require.Len(t, slice, 1)
	require.Equal(t, "ssh-banner", slice[0])
}

func TestOrchestrator_TypedWrite_SinglePublishWhenSchemaRegistered(t *testing.T) {
	dag := &DAGDefinition{Name: "typed-single", Nodes: []DAGNodeConfig{{InstanceID: "n1", ModuleType: "test-min2"}}}

	RegisterModuleFactory("test-min2", func() Module {
		return &minimalTestModule{
			meta: ModuleMetadata{
				Name:        "test-min2",
				Description: "emits one output",
				Produces:    []DataContractEntry{{Key: "config.targets", Cardinality: CardinalitySingle}},
			},
			outKey:  "config.targets",
			outData: []string{"10.0.0.1"},
		}
	})

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	// Register schema for single key: []string
	err = orc.dataCtx.RegisterType("config.targets", reflect.TypeOf([]string{}), CardinalitySingle)
	require.NoError(t, err)

	_, runErr := orc.Run(context.Background(), nil)
	require.NoError(t, runErr)

	v, err := orc.dataCtx.GetValue("config.targets")
	require.NoError(t, err)
	targets, ok := v.([]string)
	require.True(t, ok)
	require.Equal(t, []string{"10.0.0.1"}, targets)
}

func TestOrchestrator_TypedWrite_FallbackWhenUnregistered(t *testing.T) {
	dag := &DAGDefinition{Name: "legacy-fallback", Nodes: []DAGNodeConfig{{InstanceID: "n1", ModuleType: "test-min3"}}}

	RegisterModuleFactory("test-min3", func() Module {
		return &minimalTestModule{
			meta:    ModuleMetadata{Produces: []DataContractEntry{{Key: "unregistered.key", Cardinality: CardinalityList}}},
			outKey:  "unregistered.key",
			outData: 123,
		}
	})

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	// No schema registration here on purpose
	_, runErr := orc.Run(context.Background(), nil)
	require.NoError(t, runErr)

	// Legacy path stores []interface{}
	all := orc.dataCtx.GetAll()
	v, ok := all["unregistered.key"]
	require.True(t, ok)
	list, ok := v.([]interface{})
	require.True(t, ok)
	require.Len(t, list, 1)
	require.Equal(t, 123, list[0])
}
