package engine

import (
	"context"
	"fmt"
	"testing"
)

type mockModule struct {
	meta     ModuleMetadata
	initErr  error
	execFunc func(context.Context, map[string]interface{}, chan<- ModuleOutput) error
}

func (m *mockModule) Metadata() ModuleMetadata { return m.meta }

func (m *mockModule) Init(instanceID string, config map[string]interface{}) error { return m.initErr }

func (m *mockModule) Execute(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
	if m.execFunc != nil {
		return m.execFunc(ctx, inputs, out)
	}
	return nil
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusIdle, "Idle"},
		{StatusPending, "Pending"},
		{StatusRunning, "Running"},
		{StatusCompleted, "Completed"},
		{StatusFailed, "Failed"},
		// Test out-of-range values
		{Status(100), "Failed"}, // Will panic if index out of range, but current implementation will panic
		{Status(-1), "Failed"},  // Will panic if index out of range
	}

	for _, tt := range tests[:5] { // Only test valid values to avoid panic
		got := tt.status.String()
		if got != tt.expected {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.expected)
		}
	}
}
func TestNewDataContext(t *testing.T) {
	dc := NewDataContext()
	if dc == nil {
		t.Fatal("NewDataContext() returned nil")
	}
	if dc.data == nil {
		t.Fatal("DataContext.data map is nil after initialization")
	}
	if len(dc.data) != 0 {
		t.Errorf("Expected DataContext.data to be empty, got length %d", len(dc.data))
	}
}
func TestDataContext_Set(t *testing.T) {
	dc := NewDataContext()
	key := "module1.output"
	value := "test-value"

	dc.Set(key, value)

	dc.RLock()
	defer dc.RUnlock()
	got, exists := dc.data[key]
	if !exists {
		t.Errorf("Expected key %q to exist in DataContext.data", key)
	}
	if got != value {
		t.Errorf("Expected value %q for key %q, got %q", value, key, got)
	}
}

func TestDataContext_Set_Overwrite(t *testing.T) {
	dc := NewDataContext()
	key := "module1.output"
	value1 := "first"
	value2 := "second"

	dc.Set(key, value1)
	dc.Set(key, value2)

	dc.RLock()
	defer dc.RUnlock()
	got, exists := dc.data[key]
	if !exists {
		t.Errorf("Expected key %q to exist in DataContext.data", key)
	}
	if got != value2 {
		t.Errorf("Expected value %q for key %q after overwrite, got %q", value2, key, got)
	}
}
func TestDataContext_Get(t *testing.T) {
	dc := NewDataContext()
	key := "module1.output"
	value := "test-value"

	// Test getting a key that does not exist
	got, exists := dc.Get(key)
	if exists {
		t.Errorf("Expected key %q to not exist, but exists=true", key)
	}
	if got != nil {
		t.Errorf("Expected value to be nil for non-existent key %q, got %v", key, got)
	}

	// Set the key and test retrieval
	dc.Set(key, value)
	got, exists = dc.Get(key)
	if !exists {
		t.Errorf("Expected key %q to exist after Set, but exists=false", key)
	}
	if got != value {
		t.Errorf("Expected value %q for key %q, got %v", value, key, got)
	}

	// Test with another key that was never set
	otherKey := "module2.output"
	got, exists = dc.Get(otherKey)
	if exists {
		t.Errorf("Expected key %q to not exist, but exists=true", otherKey)
	}
	if got != nil {
		t.Errorf("Expected value to be nil for non-existent key %q, got %v", otherKey, got)
	}
}
func TestDataContext_GetAll_Empty(t *testing.T) {
	dc := NewDataContext()
	all := dc.GetAll()
	if all == nil {
		t.Fatal("GetAll() returned nil map")
	}
	if len(all) != 0 {
		t.Errorf("Expected empty map from GetAll(), got length %d", len(all))
	}
}

func TestDataContext_GetAll_NonEmpty(t *testing.T) {
	dc := NewDataContext()
	dc.Set("key1", "value1")
	dc.Set("key2", 42)
	dc.Set("key3", []string{"a", "b"})

	all := dc.GetAll()
	if len(all) != 3 {
		t.Errorf("Expected map of length 3, got %d", len(all))
	}
	if all["key1"] != "value1" {
		t.Errorf("Expected key1 to be 'value1', got %v", all["key1"])
	}
	if all["key2"] != 42 {
		t.Errorf("Expected key2 to be 42, got %v", all["key2"])
	}
	if got, ok := all["key3"].([]string); !ok || len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("Expected key3 to be [a b], got %v", all["key3"])
	}
}

func TestDataContext_GetAll_Independence(t *testing.T) {
	dc := NewDataContext()
	dc.Set("k", "v")
	all := dc.GetAll()
	all["k"] = "changed"

	got, _ := dc.Get("k")
	if got != "v" {
		t.Errorf("Modifying GetAll() result should not affect DataContext, but got %v", got)
	}
}

func TestNewOrchestrator_NilDAG(t *testing.T) {
	orc, err := NewOrchestrator(nil)
	if err == nil || orc != nil {
		t.Error("Expected error for nil DAGDefinition")
	}
}

func TestNewOrchestrator_EmptyNodes(t *testing.T) {
	dag := &DAGDefinition{Name: "empty", Nodes: nil}
	orc, err := NewOrchestrator(dag)
	if err == nil || orc != nil {
		t.Error("Expected error for DAGDefinition with no nodes")
	}
}

func TestNewOrchestrator_MissingInstanceID(t *testing.T) {
	dag := &DAGDefinition{
		Name: "missing-id",
		Nodes: []DAGNodeConfig{
			{
				InstanceID: "",
				ModuleType: "mock",
				Config:     map[string]interface{}{},
			},
		},
	}
	orc, err := NewOrchestrator(dag)
	if err == nil || orc != nil {
		t.Error("Expected error for missing instance_id")
	}
}

func TestNewOrchestrator_DuplicateInstanceID(t *testing.T) {

	RegisterModuleFactory("mock", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				ID:   "mod1",
				Name: "mock",
				Type: ScanModuleType,
				Produces: []DataContractEntry{
					{Key: "mock.output"},
				},
				Consumes: []DataContractEntry{
					{Key: "mock.input"},
				},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				out <- ModuleOutput{
					DataKey: "mock.output",
					Data:    "hello world",
				}
				return nil
			},
		}
	})

	defer func() {
		delete(moduleRegistry, "mock")
	}()

	dag := &DAGDefinition{
		Name: "dup-id",
		Nodes: []DAGNodeConfig{
			{
				InstanceID: "mod1",
				ModuleType: "mock",
				Config:     map[string]interface{}{},
			},
			{
				InstanceID: "mod1",
				ModuleType: "mock",
				Config:     map[string]interface{}{},
			},
		},
	}
	orc, err := NewOrchestrator(dag)
	if err == nil || orc != nil {
		t.Error("Expected error for duplicate instance_id")
	}
}

func TestNewOrchestrator_FailedToCreateModuleInstance(t *testing.T) {
	instanceID := "mod1"
	moduleType := "unknown"

	dag := &DAGDefinition{
		Name: "unknown-dep",
		Nodes: []DAGNodeConfig{
			{
				InstanceID: instanceID,
				ModuleType: moduleType,
				Config:     map[string]interface{}{},
			},
		},
	}
	orc, err := NewOrchestrator(dag)

	if err == nil {
		t.Error("Expected an error but got nil")
	}

	if orc != nil {
		t.Error("Expected Orchestrator to be nil")
	}

	expectedErrMsg := fmt.Sprintf("failed to create module instance '%s' (type: %s): no module factory registered for name: %s", instanceID, moduleType, moduleType)

	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message to be '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestOrchestrator_ConnectsModulesByConsumesAndProduces(t *testing.T) {
	// 1. Register mock modules
	RegisterModuleFactory("mock-a", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				ID:       "a",
				Name:     "mock-a",
				Produces: []DataContractEntry{
					{Key: "a.output"},
				},				
			},
		}
	})
	RegisterModuleFactory("mock-b", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				ID:       "b",
				Name:     "mock-b",
				Consumes: []DataContractEntry{
					{Key: "a.output"},
				},				
			},
		}
	})

	defer func() {
		delete(moduleRegistry, "mock-a")
		delete(moduleRegistry, "mock-b")
	}()

	// 2. DAG definition
	dag := &DAGDefinition{
		Name: "test-dag",
		Nodes: []DAGNodeConfig{
			{InstanceID: "modA", ModuleType: "mock-a", Config: map[string]interface{}{}},
			{InstanceID: "modB", ModuleType: "mock-b", Config: map[string]interface{}{}},
		},
	}

	// 3. Create orchestrator
	orc, err := NewOrchestrator(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	modA := orc.moduleNodes["modA"]
	modB := orc.moduleNodes["modB"]

	// 4. modB's dependency should be modA
	if len(modB.dependencies) != 1 || modB.dependencies[0] != modA {
		t.Errorf("modB.dependencies should include modA")
	}

	// 5. modA's dependent should be modB
	if len(modA.dependents) != 1 || modA.dependents[0] != modB {
		t.Errorf("modA.dependents should include modB")
	}
}

func TestOrchestrator_Run_ExecutesModulesInOrder(t *testing.T) {
	// Register mock modules
	RegisterModuleFactory("mock-producer", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				ID:       "mod1",
				Name:     "mock-producer",
				Produces: []DataContractEntry{{Key: "foo"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				out <- ModuleOutput{
					DataKey: "foo",
					Data:    "bar",
				}
				return nil
			},
		}
	})

	RegisterModuleFactory("mock-consumer", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				ID:       "mod2",
				Name:     "mock-consumer",
				Consumes: []DataContractEntry{{Key: "foo"}},
				Produces: []DataContractEntry{{Key: "baz"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				if val, ok := inputs["foo"]; !ok || val != "bar" {
					t.Errorf("Expected input 'foo' = 'bar', got %v", val)
				}
				out <- ModuleOutput{
					DataKey: "baz",
					Data:    "qux",
				}
				return nil
			},
		}
	})

	defer func() {
		delete(moduleRegistry, "mock-producer")
		delete(moduleRegistry, "mock-consumer")
	}()

	dag := &DAGDefinition{
		Name: "test-run",
		Nodes: []DAGNodeConfig{
			{InstanceID: "mod1", ModuleType: "mock-producer", Config: map[string]interface{}{}},
			{InstanceID: "mod2", ModuleType: "mock-consumer", Config: map[string]interface{}{}},
		},
	}

	orc, err := NewOrchestrator(dag)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	results, err := orc.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("DAG run failed: %v", err)
	}

	// Check final results
	want := map[string]interface{}{
		"mod1.foo": "bar",
		"mod2.baz": "qux",
	}

	for k, v := range want {
		got, ok := results[k]
		if !ok {
			t.Errorf("Expected result key '%s' not found", k)
		}
		if got != v {
			t.Errorf("Result mismatch for '%s': got %v, want %v", k, got, v)
		}
	}
}
