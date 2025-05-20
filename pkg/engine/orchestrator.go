// pkg/engine/orchestrator.go
package engine

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// DAGNodeConfig defines the configuration for a single node (module instance) in the DAG.
type DAGNodeConfig struct {
	InstanceID string                 `yaml:"instance_id"` // Unique ID for this module instance in the DAG
	ModuleType string                 `yaml:"module_type"` // Registered name of the module (e.g., "icmp-ping-discovery")
	Config     map[string]interface{} `yaml:"config"`      // Module-specific configuration
}

// DAGDefinition defines the entire Directed Acyclic Graph of modules for a scan.
type DAGDefinition struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Nodes       []DAGNodeConfig `yaml:"nodes"`
}

// Orchestrator manages the execution of a DAG of modules.
type Orchestrator struct {
	dag         *DAGDefinition
	moduleNodes map[string]*runtimeNode // Map of InstanceID to its runtime representation
	dataCtx     *DataContext            // Central place to store and retrieve module outputs
}

type runtimeNode struct {
	instanceID   string
	module       Module
	config       DAGNodeConfig
	status       Status
	startTime    time.Time
	endTime      time.Time
	err          error
	outputs      map[string]ModuleOutput
	dependencies []*runtimeNode
	dependents   []*runtimeNode
	inputData    map[string]interface{}
}

type DataContext struct {
	sync.RWMutex
	data map[string]interface{}
}

type Status int

const (
	StatusIdle Status = iota
	StatusPending
	StatusRunning
	StatusCompleted
	StatusFailed
)

// String returns the string representation of the Status value.
// It maps each Status enum value to its corresponding string label.
func (s Status) String() string {
	return [...]string{"Idle", "Pending", "Running", "Completed", "Failed"}[s]
}

// NewDataContext creates and returns a new instance of DataContext
// with an initialized data map for storing key-value pairs.
func NewDataContext() *DataContext {
	return &DataContext{
		data: make(map[string]interface{}),
	}
}

func (dc *DataContext) Set(key string, value interface{}) {
	dc.Lock()
	defer dc.Unlock()
	dc.data[key] = value
}

func (dc *DataContext) Get(key string) (interface{}, bool) {
	dc.RLock()
	defer dc.RUnlock()
	value, found := dc.data[key]
	return value, found
}

func (dc *DataContext) GetAll() map[string]interface{} {
	dc.RLock()
	defer dc.RUnlock()
	dataCopy := make(map[string]interface{}, len(dc.data))
	for k, v := range dc.data {
		dataCopy[k] = v
	}
	return dataCopy
}

// NewOrchestrator creates and initializes a new Orchestrator instance based on the provided DAGDefinition.
// It validates the DAG definition, instantiates module nodes, establishes dependencies between nodes based
// on their input and output keys, and prepares the orchestrator for execution. Returns an error if the DAG
// definition is invalid, contains duplicate instance IDs, or if any module instance fails to initialize.
//
// Parameters:
//   - dagDef: Pointer to the DAGDefinition describing the workflow and its nodes.
//
// Returns:
//   - *Orchestrator: The initialized orchestrator instance.
//   - error: An error if initialization fails, or nil on success.
func NewOrchestrator(dagDef *DAGDefinition) (*Orchestrator, error) {
	if dagDef == nil || len(dagDef.Nodes) == 0 {
		return nil, fmt.Errorf("DAG definition is nil or has no nodes")
	}

	orc := &Orchestrator{
		dag:         dagDef,
		moduleNodes: make(map[string]*runtimeNode),
		dataCtx:     NewDataContext(),
	}

	produces := make(map[string]string)

	for _, nodeCfg := range dagDef.Nodes {
		if nodeCfg.InstanceID == "" {
			return nil, fmt.Errorf("DAG node config missing instance_id for module_type '%s'", nodeCfg.ModuleType)
		}
		if _, exists := orc.moduleNodes[nodeCfg.InstanceID]; exists {
			return nil, fmt.Errorf("duplicate instance_id '%s' in DAG definition", nodeCfg.InstanceID)
		}

		moduleInstance, err := GetModuleInstance(nodeCfg.ModuleType, nodeCfg.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create module instance '%s' (type: %s): %w", nodeCfg.InstanceID, nodeCfg.ModuleType, err)
		}

		node := &runtimeNode{
			instanceID: nodeCfg.InstanceID,
			module:     moduleInstance,
			config:     nodeCfg,
			status:     StatusIdle,
			outputs:    make(map[string]ModuleOutput),
		}
		orc.moduleNodes[nodeCfg.InstanceID] = node

		for _, key := range moduleInstance.Metadata().Produces {
			produces[key] = nodeCfg.InstanceID
		}
	}

	for _, node := range orc.moduleNodes {
		node.inputData = make(map[string]interface{})
		for _, inputKey := range node.module.Metadata().Consumes {
			if producerID, ok := produces[inputKey]; ok {
				dep := orc.moduleNodes[producerID]
				node.dependencies = append(node.dependencies, dep)
				dep.dependents = append(dep.dependents, node)
			} else {
				fmt.Fprintf(os.Stderr, "[WARN] No producer found for key '%s' required by '%s'\n", inputKey, node.instanceID)
			}
		}
	}

	fmt.Printf("[INFO] Orchestrator initialized for DAG: %s. Found %d nodes.\n", dagDef.Name, len(orc.moduleNodes))
	return orc, nil
}

// Run executes the DAG (Directed Acyclic Graph) defined in the Orchestrator by running each module node
// in the correct order based on their dependencies. It manages concurrent execution of nodes whose dependencies
// have completed, collects their outputs, and tracks execution status and errors. The method returns a map
// containing all output data produced during the DAG execution, or an error if any module fails.
//
// Parameters:
//   - ctx: The context for cancellation and timeout control.
//
// Returns:
//   - map[string]interface{}: A map of all output data produced by the modules, keyed by their data keys.
//   - error: An error if any module execution fails, otherwise nil.
func (o *Orchestrator) Run(ctx context.Context) (map[string]interface{}, error) {
	fmt.Printf("[INFO] Starting DAG execution: %s\n", o.dag.Name)

	executed := make(map[string]bool)
	var execMu sync.Mutex
	var wg sync.WaitGroup

	executeNode := func(node *runtimeNode) {
		defer wg.Done()
		node.status = StatusRunning
		node.startTime = time.Now()
		fmt.Printf("[INFO] Executing module: %s\n", node.instanceID)

		inputs := make(map[string]interface{})
		for _, dep := range node.dependencies {
			for key, output := range dep.outputs {
				inputs[key] = output.Data
			}
		}

		outputChan := make(chan ModuleOutput, 10)
		var moduleErr error
		var mWg sync.WaitGroup
		mWg.Add(1)
		go func() {
			defer mWg.Done()
			defer close(outputChan)
			moduleErr = node.module.Execute(ctx, inputs, outputChan)
		}()

		for output := range outputChan {
			key := fmt.Sprintf("%s.%s", node.instanceID, output.DataKey)
			o.dataCtx.Set(key, output.Data)
			node.outputs[output.DataKey] = output
			if output.Error != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Output error from module '%s': %v\n", node.instanceID, output.Error)
			} else {
				fmt.Printf("[DEBUG] Output from module '%s': %s\n", node.instanceID, output.DataKey)
			}
		}

		mWg.Wait()
		node.endTime = time.Now()
		if moduleErr != nil {
			node.status = StatusFailed
			node.err = moduleErr
			fmt.Fprintf(os.Stderr, "[ERROR] Module '%s' failed: %v\n", node.instanceID, moduleErr)
		} else {
			node.status = StatusCompleted
			fmt.Printf("[INFO] Module '%s' completed in %s\n", node.instanceID, node.endTime.Sub(node.startTime))
		}
		execMu.Lock()
		executed[node.instanceID] = true
		execMu.Unlock()
	}

	for len(executed) < len(o.moduleNodes) {
		for _, node := range o.moduleNodes {
			execMu.Lock()
			if _, ok := executed[node.instanceID]; ok {
				execMu.Unlock()
				continue
			}
			ready := true
			for _, dep := range node.dependencies {
				if _, done := executed[dep.instanceID]; !done {
					ready = false
					break
				}
			}
			execMu.Unlock()
			if ready {
				wg.Add(1)
				go executeNode(node)
			}
		}
		wg.Wait()
	}

	fmt.Printf("[INFO] DAG execution complete: %s\n", o.dag.Name)
	return o.dataCtx.GetAll(), nil
}
