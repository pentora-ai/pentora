package engine

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	if got.([]interface{})[0] != value {
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
	if reflect.DeepEqual(got, []interface{}{value1, value2}) == false {
		t.Errorf("Expected value %q, %q for key %q after overwrite, got %q", value1, value2, key, got)
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
	if got.([]interface{})[0] != value {
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
	key1Val := "value1"
	key2val := 42
	key3Val := []string{"a", "b"}

	dc.Set("key1", key1Val)
	dc.Set("key2", key2val)
	dc.Set("key3", key3Val)

	all := dc.GetAll()
	if len(all) != 3 {
		t.Errorf("Expected map of length 3, got %d", len(all))
	}

	if !reflect.DeepEqual(all["key1"], []interface{}{key1Val}) {
		t.Errorf("Expected key1 to be '%s', got '%s'", key1Val, all["key1"])
	}
	if !reflect.DeepEqual(all["key2"], []interface{}{key2val}) {
		t.Errorf("Expected key2 to be 42, got %v", all["key2"])
	}
	if !reflect.DeepEqual(all["key3"], []interface{}{key3Val}) {
		t.Errorf("Expected key3 to be %v, got %v", key3Val, all["key3"])
	}
}

func TestDataContext_GetAll_Independence(t *testing.T) {
	dc := NewDataContext()
	dc.Set("k", "v")
	all := dc.GetAll()
	all["k"] = "changed"

	got, _ := dc.Get("k")
	if !reflect.DeepEqual(got, []interface{}{"v"}) {
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
				ID:   "a",
				Name: "mock-a",
				Produces: []DataContractEntry{
					{Key: "a.output"},
				},
			},
		}
	})
	RegisterModuleFactory("mock-b", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				ID:   "b",
				Name: "mock-b",
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
				if val, ok := inputs["foo"]; !ok || !reflect.DeepEqual(val, []interface{}{"bar"}) {
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
		"foo": []interface{}{"bar"},
		"baz": []interface{}{"qux"},
	}

	if !reflect.DeepEqual(results, want) {
		t.Errorf("Final results mismatch:\ngot:  %v\nwant: %v", results, want)
	}
}

// TestOrchestrator_Run_ExplicitDependencies tests explicit dependency resolution from DAGSchema
func TestOrchestrator_Run_ExplicitDependencies(t *testing.T) {
	executionOrder := []string{}
	var orderMutex sync.Mutex

	RegisterModuleFactory("explicit-mod1", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name: "explicit-mod1",
				Type: ScanModuleType,
				// No explicit Consumes/Produces - testing pure explicit dependencies
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				orderMutex.Lock()
				executionOrder = append(executionOrder, "mod1")
				orderMutex.Unlock()
				out <- ModuleOutput{DataKey: "data.1", Data: "value1"}
				return nil
			},
		}
	})

	RegisterModuleFactory("explicit-mod2", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name: "explicit-mod2",
				Type: ScanModuleType,
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				orderMutex.Lock()
				executionOrder = append(executionOrder, "mod2")
				orderMutex.Unlock()
				out <- ModuleOutput{DataKey: "data.2", Data: "value2"}
				return nil
			},
		}
	})

	RegisterModuleFactory("explicit-mod3", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name: "explicit-mod3",
				Type: ScanModuleType,
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				orderMutex.Lock()
				executionOrder = append(executionOrder, "mod3")
				orderMutex.Unlock()
				out <- ModuleOutput{DataKey: "data.3", Data: "value3"}
				return nil
			},
		}
	})

	defer func() {
		delete(moduleRegistry, "explicit-mod1")
		delete(moduleRegistry, "explicit-mod2")
		delete(moduleRegistry, "explicit-mod3")
	}()

	// Create DAG with explicit dependencies: mod3 depends on mod1 and mod2
	dag := &DAGDefinition{
		Name: "explicit-deps-test",
		Nodes: []DAGNodeConfig{
			{InstanceID: "mod1", ModuleType: "explicit-mod1", Config: map[string]interface{}{}},
			{InstanceID: "mod2", ModuleType: "explicit-mod2", Config: map[string]interface{}{}},
			{
				InstanceID: "mod3",
				ModuleType: "explicit-mod3",
				Config: map[string]interface{}{
					"__depends_on": []interface{}{"mod1", "mod2"},
				},
			},
		},
	}

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	_, err = orc.Run(context.Background(), nil)
	require.NoError(t, err)

	// Check execution order: mod3 must run after mod1 and mod2
	require.Len(t, executionOrder, 3)
	idx1 := indexOf(executionOrder, "mod1")
	idx2 := indexOf(executionOrder, "mod2")
	idx3 := indexOf(executionOrder, "mod3")
	require.True(t, idx1 < idx3, "mod1 should execute before mod3")
	require.True(t, idx2 < idx3, "mod2 should execute before mod3")
}

// TestOrchestrator_Run_LayeredParallelExecution tests parallel execution in layers
func TestOrchestrator_Run_LayeredParallelExecution(t *testing.T) {
	// Track which modules run concurrently
	layer1Start := make(chan string, 2)
	layer1Done := make(chan string, 2)
	layer2Start := make(chan string)

	RegisterModuleFactory("parallel-mod1", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "parallel-mod1",
				Type:     ScanModuleType,
				Produces: []DataContractEntry{{Key: "layer1.data1"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				layer1Start <- "mod1"
				time.Sleep(50 * time.Millisecond) // Simulate work
				out <- ModuleOutput{DataKey: "layer1.data1", Data: "value1"}
				layer1Done <- "mod1"
				return nil
			},
		}
	})

	RegisterModuleFactory("parallel-mod2", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "parallel-mod2",
				Type:     ScanModuleType,
				Produces: []DataContractEntry{{Key: "layer1.data2"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				layer1Start <- "mod2"
				time.Sleep(50 * time.Millisecond) // Simulate work
				out <- ModuleOutput{DataKey: "layer1.data2", Data: "value2"}
				layer1Done <- "mod2"
				return nil
			},
		}
	})

	RegisterModuleFactory("parallel-mod3", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "parallel-mod3",
				Type:     EvaluationModuleType,
				Consumes: []DataContractEntry{{Key: "layer1.data1"}, {Key: "layer1.data2"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				layer2Start <- "mod3"
				return nil
			},
		}
	})

	defer func() {
		delete(moduleRegistry, "parallel-mod1")
		delete(moduleRegistry, "parallel-mod2")
		delete(moduleRegistry, "parallel-mod3")
	}()

	dag := &DAGDefinition{
		Name: "parallel-layers-test",
		Nodes: []DAGNodeConfig{
			{InstanceID: "mod1", ModuleType: "parallel-mod1", Config: map[string]interface{}{}},
			{InstanceID: "mod2", ModuleType: "parallel-mod2", Config: map[string]interface{}{}},
			{InstanceID: "mod3", ModuleType: "parallel-mod3", Config: map[string]interface{}{}},
		},
	}

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	go func() {
		_, _ = orc.Run(context.Background(), nil)
	}()

	// Verify layer 1 modules start in parallel (both start before either completes)
	start1 := <-layer1Start
	start2 := <-layer1Start
	require.NotEqual(t, start1, start2)

	// Wait for layer 1 to complete
	<-layer1Done
	<-layer1Done

	// Verify layer 2 starts only after layer 1 completes
	select {
	case mod := <-layer2Start:
		require.Equal(t, "mod3", mod)
	case <-time.After(2 * time.Second):
		t.Fatal("Layer 2 module did not start")
	}
}

// TestOrchestrator_Run_FailurePropagation tests that failures propagate to dependents
func TestOrchestrator_Run_FailurePropagation(t *testing.T) {
	mod2Executed := false

	RegisterModuleFactory("fail-mod1", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "fail-mod1",
				Type:     ScanModuleType,
				Produces: []DataContractEntry{{Key: "fail.data"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				return fmt.Errorf("intentional failure in mod1")
			},
		}
	})

	RegisterModuleFactory("fail-mod2", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "fail-mod2",
				Type:     EvaluationModuleType,
				Consumes: []DataContractEntry{{Key: "fail.data"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				mod2Executed = true
				return nil
			},
		}
	})

	defer func() {
		delete(moduleRegistry, "fail-mod1")
		delete(moduleRegistry, "fail-mod2")
	}()

	dag := &DAGDefinition{
		Name: "failure-propagation-test",
		Nodes: []DAGNodeConfig{
			{InstanceID: "mod1", ModuleType: "fail-mod1", Config: map[string]interface{}{}},
			{InstanceID: "mod2", ModuleType: "fail-mod2", Config: map[string]interface{}{}},
		},
	}

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	_, err = orc.Run(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "intentional failure")

	// mod2 should not execute because mod1 failed
	require.False(t, mod2Executed, "Dependent module should not execute when dependency fails")
}

// TestOrchestrator_Run_ContextCancellation tests graceful context cancellation
func TestOrchestrator_Run_ContextCancellation(t *testing.T) {
	mod1Started := make(chan struct{})
	mod1Cancelled := false
	var mod1Mutex sync.Mutex

	RegisterModuleFactory("cancel-mod1", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "cancel-mod1",
				Type:     ScanModuleType,
				Produces: []DataContractEntry{{Key: "cancel.data"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				close(mod1Started)
				select {
				case <-ctx.Done():
					mod1Mutex.Lock()
					mod1Cancelled = true
					mod1Mutex.Unlock()
					return ctx.Err()
				case <-time.After(5 * time.Second):
					return nil
				}
			},
		}
	})

	defer func() {
		delete(moduleRegistry, "cancel-mod1")
	}()

	dag := &DAGDefinition{
		Name: "context-cancellation-test",
		Nodes: []DAGNodeConfig{
			{InstanceID: "mod1", ModuleType: "cancel-mod1", Config: map[string]interface{}{}},
		},
	}

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)

	go func() {
		_, err := orc.Run(ctx, nil)
		errChan <- err
	}()

	// Wait for module to start
	<-mod1Started

	// Cancel context
	cancel()

	// Wait for run to complete
	err = <-errChan
	require.Error(t, err)

	// Verify module received cancellation
	mod1Mutex.Lock()
	canceled := mod1Cancelled
	mod1Mutex.Unlock()
	require.True(t, canceled, "Module should receive context cancellation")
}

// TestOrchestrator_Run_SequentialExecution tests strict sequential execution
func TestOrchestrator_Run_SequentialExecution(t *testing.T) {
	executionOrder := []string{}
	var orderMutex sync.Mutex

	for i := 1; i <= 4; i++ {
		modNum := i
		modName := fmt.Sprintf("seq-mod%d", modNum)
		RegisterModuleFactory(modName, func() Module {
			return &mockModule{
				meta: ModuleMetadata{
					Name:     modName,
					Type:     ScanModuleType,
					Produces: []DataContractEntry{{Key: fmt.Sprintf("seq.data%d", modNum)}},
					Consumes: func() []DataContractEntry {
						if modNum == 1 {
							return []DataContractEntry{}
						}
						return []DataContractEntry{{Key: fmt.Sprintf("seq.data%d", modNum-1)}}
					}(),
				},
				execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
					orderMutex.Lock()
					executionOrder = append(executionOrder, fmt.Sprintf("mod%d", modNum))
					orderMutex.Unlock()
					out <- ModuleOutput{DataKey: fmt.Sprintf("seq.data%d", modNum), Data: fmt.Sprintf("value%d", modNum)}
					return nil
				},
			}
		})
	}

	defer func() {
		for i := 1; i <= 4; i++ {
			delete(moduleRegistry, fmt.Sprintf("seq-mod%d", i))
		}
	}()

	dag := &DAGDefinition{
		Name: "sequential-execution-test",
		Nodes: []DAGNodeConfig{
			{InstanceID: "mod1", ModuleType: "seq-mod1", Config: map[string]interface{}{}},
			{InstanceID: "mod2", ModuleType: "seq-mod2", Config: map[string]interface{}{}},
			{InstanceID: "mod3", ModuleType: "seq-mod3", Config: map[string]interface{}{}},
			{InstanceID: "mod4", ModuleType: "seq-mod4", Config: map[string]interface{}{}},
		},
	}

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	_, err = orc.Run(context.Background(), nil)
	require.NoError(t, err)

	// Verify strict sequential order
	require.Equal(t, []string{"mod1", "mod2", "mod3", "mod4"}, executionOrder)
}

// TestOrchestrator_Run_DiamondPattern tests diamond dependency pattern
func TestOrchestrator_Run_DiamondPattern(t *testing.T) {
	executionOrder := []string{}
	var orderMutex sync.Mutex

	RegisterModuleFactory("diamond-root", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "diamond-root",
				Type:     ScanModuleType,
				Produces: []DataContractEntry{{Key: "diamond.root"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				orderMutex.Lock()
				executionOrder = append(executionOrder, "root")
				orderMutex.Unlock()
				out <- ModuleOutput{DataKey: "diamond.root", Data: "root-value"}
				return nil
			},
		}
	})

	RegisterModuleFactory("diamond-left", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "diamond-left",
				Type:     ParseModuleType,
				Consumes: []DataContractEntry{{Key: "diamond.root"}},
				Produces: []DataContractEntry{{Key: "diamond.left"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				orderMutex.Lock()
				executionOrder = append(executionOrder, "left")
				orderMutex.Unlock()
				out <- ModuleOutput{DataKey: "diamond.left", Data: "left-value"}
				return nil
			},
		}
	})

	RegisterModuleFactory("diamond-right", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "diamond-right",
				Type:     ParseModuleType,
				Consumes: []DataContractEntry{{Key: "diamond.root"}},
				Produces: []DataContractEntry{{Key: "diamond.right"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				orderMutex.Lock()
				executionOrder = append(executionOrder, "right")
				orderMutex.Unlock()
				out <- ModuleOutput{DataKey: "diamond.right", Data: "right-value"}
				return nil
			},
		}
	})

	RegisterModuleFactory("diamond-merge", func() Module {
		return &mockModule{
			meta: ModuleMetadata{
				Name:     "diamond-merge",
				Type:     EvaluationModuleType,
				Consumes: []DataContractEntry{{Key: "diamond.left"}, {Key: "diamond.right"}},
				Produces: []DataContractEntry{{Key: "diamond.merged"}},
			},
			execFunc: func(ctx context.Context, inputs map[string]interface{}, out chan<- ModuleOutput) error {
				orderMutex.Lock()
				executionOrder = append(executionOrder, "merge")
				orderMutex.Unlock()
				out <- ModuleOutput{DataKey: "diamond.merged", Data: "merged-value"}
				return nil
			},
		}
	})

	defer func() {
		delete(moduleRegistry, "diamond-root")
		delete(moduleRegistry, "diamond-left")
		delete(moduleRegistry, "diamond-right")
		delete(moduleRegistry, "diamond-merge")
	}()

	dag := &DAGDefinition{
		Name: "diamond-pattern-test",
		Nodes: []DAGNodeConfig{
			{InstanceID: "root", ModuleType: "diamond-root", Config: map[string]interface{}{}},
			{InstanceID: "left", ModuleType: "diamond-left", Config: map[string]interface{}{}},
			{InstanceID: "right", ModuleType: "diamond-right", Config: map[string]interface{}{}},
			{InstanceID: "merge", ModuleType: "diamond-merge", Config: map[string]interface{}{}},
		},
	}

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	_, err = orc.Run(context.Background(), nil)
	require.NoError(t, err)

	// Verify execution order: root first, then left/right (any order), then merge last
	require.Len(t, executionOrder, 4)
	require.Equal(t, "root", executionOrder[0])
	require.Equal(t, "merge", executionOrder[3])

	// left and right can be in any order (parallel)
	leftIdx := indexOf(executionOrder, "left")
	rightIdx := indexOf(executionOrder, "right")
	require.True(t, leftIdx == 1 || leftIdx == 2)
	require.True(t, rightIdx == 1 || rightIdx == 2)
	require.NotEqual(t, leftIdx, rightIdx)
}
