package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testLifecycleModule is a stub implementing Module and ModuleLifecycle for testing.
type testLifecycleModule struct {
	id        string
	calls     *[]string
	mu        *sync.Mutex
	failInit  bool
	failStart bool
	execDelay time.Duration
}

func (m *testLifecycleModule) Metadata() ModuleMetadata {
	return ModuleMetadata{Name: "test.lifecycle", Type: ParseModuleType}
}

func (m *testLifecycleModule) Init(instanceID string, _ map[string]interface{}) error {
	m.id = instanceID
	return nil
}

func (m *testLifecycleModule) Execute(ctx context.Context, _ map[string]interface{}, out chan<- ModuleOutput) error {
	if m.execDelay > 0 {
		time.Sleep(m.execDelay)
	}
	return nil
}

// Lifecycle
func (m *testLifecycleModule) LifecycleInit(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	*m.calls = append(*m.calls, m.id+":init")
	if m.failInit {
		return fmt.Errorf("init fail")
	}
	return nil
}

func (m *testLifecycleModule) LifecycleStart(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	*m.calls = append(*m.calls, m.id+":start")
	if m.failStart {
		return fmt.Errorf("start fail")
	}
	return nil
}

func (m *testLifecycleModule) LifecycleStop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	*m.calls = append(*m.calls, m.id+":stop")
	return nil
}

func newTestLifecycleFactory(calls *[]string, mu *sync.Mutex) ModuleFactory {
	return func() Module { return &testLifecycleModule{calls: calls, mu: mu} }
}

func TestOrchestrator_Lifecycle_OrderAndTeardown(t *testing.T) {
	// Register factories for two nodes A->B (B depends on A by consumes key)
	// To force a dependency, advertise production/consumption of a shared key
	calls := []string{}
	var mu sync.Mutex

	// Register unique names to avoid global pollution between tests
	RegisterModuleFactory("tlm-a", newTestLifecycleFactory(&calls, &mu))
	RegisterModuleFactory("tlm-b", newTestLifecycleFactory(&calls, &mu))

	// Build DAG: A then B via explicit dependency
	dag := &DAGDefinition{
		Name: "lifecycle-dag",
		Nodes: []DAGNodeConfig{
			{InstanceID: "A", ModuleType: "tlm-a", Config: map[string]interface{}{}},
			{InstanceID: "B", ModuleType: "tlm-b", Config: map[string]interface{}{"__depends_on": []string{"A"}}},
		},
	}

	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	_, err = orc.Run(context.Background(), nil)
	require.NoError(t, err)

	// Expect lifecycle order: A.init, B.init, A.start, B.start, ... stops in reverse: B.stop, A.stop
	// Execute does not add markers; we only validate lifecycle hooks and stop reverse order
	// Since goroutines are concurrent, starts may interleave but A.start should occur before B.start due to dependency
	// Validate prefix/suffix sets
	// Collect positions
	idx := func(tag string) int {
		for i, v := range calls {
			if v == tag {
				return i
			}
		}
		return -1
	}

	require.NotEqual(t, -1, idx("A:init"))
	require.NotEqual(t, -1, idx("B:init"))
	require.True(t, idx("A:init") < idx("B:init"))

	require.NotEqual(t, -1, idx("A:start"))
	require.NotEqual(t, -1, idx("B:start"))
	// Because B depends on A, A:start must precede B:start
	require.True(t, idx("A:start") < idx("B:start"))

	// Stops must be present and in reverse order: B.stop before A.stop
	require.NotEqual(t, -1, idx("A:stop"))
	require.NotEqual(t, -1, idx("B:stop"))
	require.True(t, idx("B:stop") < idx("A:stop"))
}

func TestOrchestrator_Lifecycle_StartFailureSkipsExecute(t *testing.T) {
	calls := []string{}
	var mu sync.Mutex

	RegisterModuleFactory("tlm-fail", func() Module { return &testLifecycleModule{calls: &calls, mu: &mu, failStart: true} })

	dag := &DAGDefinition{
		Name:  "lifecycle-fail",
		Nodes: []DAGNodeConfig{{InstanceID: "X", ModuleType: "tlm-fail"}},
	}
	orc, err := NewOrchestrator(dag)
	require.NoError(t, err)

	_, err = orc.Run(context.Background(), nil)
	require.Error(t, err)

	// Ensure start and stop recorded, and stop follows best-effort teardown
	idx := func(tag string) int {
		for i, v := range calls {
			if v == tag {
				return i
			}
		}
		return -1
	}
	require.NotEqual(t, -1, idx("X:init"))
	require.NotEqual(t, -1, idx("X:start"))
	require.NotEqual(t, -1, idx("X:stop"))
}
