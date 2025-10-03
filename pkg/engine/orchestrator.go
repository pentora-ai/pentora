// pkg/engine/orchestrator.go
package engine

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	logger      zerolog.Logger          // Use a logger for debug/info messages
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
	// inputData    map[string]interface{}
}

type DataContext struct {
	sync.RWMutex
	data   map[string]interface{}
	logger zerolog.Logger // Use a logger for debug/info messages
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
		data:   make(map[string]interface{}),
		logger: log.With().Str("component", "DataContext").Logger(),
	}
}

// Set adds or appends a value to the DataContext for a given key.
// All values are stored as a list ([]interface{}) to consistently handle
// single or multiple data items (e.g., multiple outputs from different targets for the same DataKey).
func (dc *DataContext) Set(key string, value interface{}) {
	dc.Lock()
	defer dc.Unlock()
	if existingData, found := dc.data[key]; found {
		if list, ok := existingData.([]interface{}); ok {
			dc.data[key] = append(list, value) // Append to existing list
		} else {
			// This case implies the key was somehow set with a non-list value directly,
			// which shouldn't happen if all writes go through this Set method.
			// Promote to a list, including the existing non-list item.
			dc.logger.Warn().Str("key", key).
				Str("existing_type", fmt.Sprintf("%T", existingData)).
				Str("new_value_type", fmt.Sprintf("%T", value)).
				Msg("Existing data for key was not a list. Promoting to list and appending new value.")
			dc.data[key] = []interface{}{existingData, value}
		}
	} else {
		// Key does not exist, create a new list with the value as its first element
		dc.data[key] = []interface{}{value}
	}

	dc.logger.Debug().Str("key", key).Str("value_type_added_to_list", fmt.Sprintf("%T", value)).Msg("Set/Appended value in DataContext")
}

// Get retrieves the value associated with the given key from the DataContext.
// It returns the value and a boolean indicating if the key was found.
func (dc *DataContext) Get(key string) (interface{}, bool) {
	dc.RLock()
	defer dc.RUnlock()
	value, found := dc.data[key]
	return value, found
}

// GetAll returns a shallow copy of all data in the context.
func (dc *DataContext) GetAll() map[string]interface{} {
	dc.RLock()
	defer dc.RUnlock()
	dataCopy := make(map[string]interface{}, len(dc.data))
	for k, v := range dc.data {
		dataCopy[k] = v
	}
	return dataCopy
}

// SetInitial stores an initial input value directly, overwriting if exists.
// Used for global inputs like "config.targets".
func (dc *DataContext) SetInitial(key string, value interface{}) {
	dc.Lock()
	defer dc.Unlock()
	dc.data[key] = value
	log.Trace().Msgf("[DEBUG-CTX-SET-INITIAL] Key: %s, Type: %T, Value: %+v", key, value, value)
}

// AddOrAppendToList adds a value to the DataContext.
// If the key doesn't exist, it creates a new list with the value.
// If the key exists and its value is a list, it appends.
// If the key exists and its value is NOT a list, it converts it to a list and appends.
// This is primarily for module outputs where multiple outputs for the same DataKey might occur.
func (dc *DataContext) AddOrAppendToList(key string, value interface{}) {
	dc.Lock()
	defer dc.Unlock()
	if existingData, found := dc.data[key]; found {
		if list, ok := existingData.([]interface{}); ok {
			dc.data[key] = append(list, value)
		} else {
			// Existing data was not a list, promote to list and append new value
			dc.data[key] = []interface{}{existingData, value}
		}
	} else {
		// Key does not exist, create a new list with the value
		dc.data[key] = []interface{}{value}
	}
	dc.logger.Trace().Str("key", key).Type("new_value_type", value).Msg("Added/Appended value to DataContext list")
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
		logger:      log.With().Str("component", "Orchestrator").Str("dag_name", dagDef.Name).Logger(),
	}

	// This map tracks which DataKey (string) is produced by which module instanceID.
	producesMap := make(map[string]string) // DataKey (string) -> producer InstanceID

	// First pass: Instantiate modules and identify what they produce.
	for _, nodeCfg := range dagDef.Nodes {
		if nodeCfg.InstanceID == "" {
			return nil, fmt.Errorf("DAG node config missing instance_id for module_type '%s'", nodeCfg.ModuleType)
		}
		if _, exists := orc.moduleNodes[nodeCfg.InstanceID]; exists {
			return nil, fmt.Errorf("duplicate instance_id '%s' in DAG definition", nodeCfg.InstanceID)
		}

		moduleInstance, err := GetModuleInstance(nodeCfg.InstanceID, nodeCfg.ModuleType, nodeCfg.Config)
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
		orc.logger.Debug().Str("instance_id", nodeCfg.InstanceID).Str("module_type", nodeCfg.ModuleType).Msg("Module instance created and initialized")

		// Register what this node produces for dependency resolution.
		// ModuleMetadata.Produces is now []DataContractEntry
		for _, contractEntry := range moduleInstance.Metadata().Produces {
			dataKeyString := contractEntry.Key // Get the string DataKey from the contract entry
			if existingProducer, found := producesMap[dataKeyString]; found {
				orc.logger.Warn().Str("data_key", dataKeyString).Str("existing_producer", existingProducer).Str("new_producer", nodeCfg.InstanceID).Msg("DataKey is produced by multiple module instances in this DAG. Dependency resolution might use the last one registered for this DataKey.")
			}
			producesMap[dataKeyString] = nodeCfg.InstanceID // Use the string DataKey
		}
	}

	// Second pass: Resolve dependencies based on Consumes and Produces metadata.
	for _, node := range orc.moduleNodes {
		nodeLogger := orc.logger.With().Str("node_instance_id", node.instanceID).Logger()
		// ModuleMetadata.Consumes is now []DataContractEntry
		for _, consumedContract := range node.module.Metadata().Consumes {
			consumedKeyString := consumedContract.Key // Get the string DataKey
			foundProducerInDAG := false

			// Check if this consumedKeyString is produced by any *other* node in this DAG.
			// The producesMap stores DataKey (string) -> producer InstanceID.
			if producerInstanceID, produced := producesMap[consumedKeyString]; produced {
				if producerNode, exists := orc.moduleNodes[producerInstanceID]; exists {
					if producerNode.instanceID != node.instanceID { // Cannot depend on itself for a key
						node.dependencies = append(node.dependencies, producerNode)
						producerNode.dependents = append(producerNode.dependents, node)
						nodeLogger.Debug().Str("consumed_key", consumedKeyString).Str("producer_instance_id", producerInstanceID).Msg("Resolved DAG dependency")
						foundProducerInDAG = true
					}
				} else {
					nodeLogger.Error().Str("consumed_key", consumedKeyString).Str("producer_id_from_map", producerInstanceID).Msg("Producer instance ID found in producesMap, but node not in moduleNodes. This is an internal error.")
				}
			}

			if !foundProducerInDAG {
				nodeLogger.Debug().Str("consumed_key", consumedKeyString).Msg("Consumed key not produced by any other DAG node; expected from initial inputs or to be optional.")
			}
		}
	}

	orc.logger.Info().Int("node_count", len(orc.moduleNodes)).Msg("Orchestrator initialized successfully")
	return orc, nil
}

// Run executes the DAG (Directed Acyclic Graph) defined in the Orchestrator by running its module nodes
// according to their dependencies. It takes a context for cancellation and an optional map of initial inputs
// to seed the global data context. The function manages concurrent execution of nodes whose dependencies are met,
// collects their outputs, and handles errors and cancellations gracefully.
//
// Parameters:
//   - ctx: context.Context for cancellation and timeout control.
//   - initialInputs: map[string]interface{} containing initial input values to be made available to modules.
//
// Returns:
//   - map[string]interface{}: A map containing all outputs collected in the global data context after execution.
//   - error: An error if any module fails or if the context is cancelled; otherwise, nil.
//
// The function ensures that:
//   - Each node is executed only after all its dependencies have completed successfully.
//   - Outputs from modules are aggregated and made available to dependent modules and the global context.
//   - Errors in module execution or dependency failures are propagated and halt further execution as appropriate.
//   - The function is safe for concurrent execution of independent nodes and uses synchronization primitives
//     to protect shared state.
func (o *Orchestrator) Run(ctx context.Context, initialInputs map[string]interface{}) (map[string]interface{}, error) {
	logger := log.With().Str("dag", o.dag.Name).Logger()
	logger.Info().Msg("Starting DAG execution")

	// Store initial inputs in the global data context
	for key, value := range initialInputs {
		o.dataCtx.SetInitial(key, value) // Use SetInitial for direct storage
	}

	// Keep track of nodes that have finished execution
	executionCompleted := make(map[string]bool)
	var completedMutex sync.Mutex // Protects executionCompleted map

	// Channel to signal that a node has finished, to re-evaluate runnable nodes
	nodeDoneSignal := make(chan string, len(o.moduleNodes))

	var overallError error
	var errorOnce sync.Once

	setOverallError := func(err error) {
		errorOnce.Do(func() {
			overallError = err
			// Optionally, here you could signal all running goroutines to cancel via a shared mechanism
			// if the context passed to Run isn't already being used for such a global cancellation.
		})
	}

	var activeGoroutines sync.WaitGroup

	// Loop until all nodes are completed or an error occurs that halts the DAG
	for len(executionCompleted) < len(o.moduleNodes) {
		madeProgressInIteration := false

		for _, node := range o.moduleNodes {
			completedMutex.Lock()
			_, alreadyCompleted := executionCompleted[node.instanceID]
			isRunningOrPending := node.status == StatusRunning || node.status == StatusPending
			completedMutex.Unlock()

			if alreadyCompleted || isRunningOrPending {
				continue
			}

			// Check if all dependencies are met
			dependenciesMet := true
			nodeInputs := make(map[string]interface{})

			// 1. Gather inputs from dependencies
			for _, dep := range node.dependencies {
				completedMutex.Lock()
				depCompleted := executionCompleted[dep.instanceID]
				depFailed := dep.status == StatusFailed
				completedMutex.Unlock()

				if !depCompleted {
					dependenciesMet = false
					break
				}
				if depFailed {
					// If a dependency failed, this node cannot run. Mark as failed/skipped.
					node.status = StatusFailed
					node.err = fmt.Errorf("dependency '%s' failed", dep.instanceID)
					fmt.Fprintf(os.Stderr, "[ERROR] Module '%s' cannot run because dependency '%s' failed.\n", node.instanceID, dep.instanceID)

					completedMutex.Lock()
					executionCompleted[node.instanceID] = true // Mark as "handled" to avoid re-processing
					completedMutex.Unlock()

					setOverallError(node.err) // Propagate error
					dependenciesMet = false   // Ensure it doesn't try to run
					break
				}
				// Collect outputs from completed dependencies
				for dataKey, output := range dep.outputs {
					// Key for data context from producer: instanceID.dataKey
					// Key for consumer: dataKey
					nodeInputs[dataKey] = []interface{}{output.Data}
				}
			}

			if !dependenciesMet {
				// if node.status == StatusFailed && overallError != nil { // If already marked as failed due to dependency
				// Already handled, just make sure to break the outer loop if an error requires halting.
				// }
				continue
			}

			// 2. Gather inputs from initial/global context for keys not provided by dependencies
			for _, consumedContract := range node.module.Metadata().Consumes {
				consumedKeyString := consumedContract.Key // Use the string Key
				if _, providedByDependency := nodeInputs[consumedKeyString]; !providedByDependency {
					// Key not provided by a direct DAG dependency, try to get it from the global/initial context
					// if val, found := o.dataCtx.Get(consumedKey); found {
					if val, found := o.dataCtx.Get(consumedKeyString); found {
						// config.targets was set by SetInitial, so it should be []string directly.
						// Module outputs (like discovery.live_hosts from icmp_ping) were set by AddOrAppendToList,
						// so they will be []interface{}.
						if consumedKeyString == "config.targets" { // Specific handling for initial inputs
							if _, ok := val.([]string); ok {
								nodeInputs[consumedKeyString] = val
								o.dataCtx.logger.Trace().Msgf("Input '%s' for '%s' from DataContext (direct as []string): %+v", consumedKeyString, node.instanceID, val)
							} else {
								log.Error().Msgf("[ERROR-ORC] Expected 'config.targets' to be []string in DataContext, got %T", val)
							}
						} else { // For other keys, assume they might be lists from module outputs
							nodeInputs[consumedKeyString] = val // Pass as is, module Execute will handle unwrapping list if needed
							log.Trace().Msgf("[DEBUG-ORC-GETINPUT] Input '%s' for '%s' from DataContext (type %T): %+v", consumedKeyString, node.instanceID, val, val)
							log.Debug().Str("node", node.instanceID).Str("input_key", consumedKeyString).Str("source_dependency", "dep.instanceID").Type("type_passed_to_module", val).Msg("Input from dependency added to nodeInputs")
						}
					} else {
						// Optional input not found, module should handle this.
						// If required and not found, module's Execute should error.
						logger.Debug().Msgf("Orchestrator: Optional input key '%s' not found in dependencies or initial context for module '%s'.", consumedKeyString, node.instanceID)
					}
				} else {
					if dataVal, dataOk := o.dataCtx.Get(consumedKeyString); dataOk {
						if dataSlice, ok1 := dataVal.([]interface{}); ok1 {
							if nodeSlice, ok2 := nodeInputs[consumedKeyString].([]interface{}); ok2 {
								if len(dataSlice) > len(nodeSlice) {
									nodeInputs[consumedKeyString] = dataVal
								}
							}
						}
					}
				}
			}

			// Launch the node
			node.status = StatusPending
			activeGoroutines.Add(1)
			madeProgressInIteration = true

			go func(currentNode *runtimeNode, inputsForNode map[string]interface{}) {
				defer activeGoroutines.Done()

				execContext, execCancel := context.WithCancel(ctx) // Create a context for this specific execution
				defer execCancel()

				mlogger := o.logger.With().
					Str("module", currentNode.instanceID).Logger()

				currentNode.startTime = time.Now()
				mlogger.Info().Msg("Executing module")

				completedMutex.Lock()
				currentNode.status = StatusRunning
				completedMutex.Unlock()

				outputChan := make(chan ModuleOutput, 10) // Buffered channel
				var moduleErr error
				var moduleWg sync.WaitGroup
				moduleWg.Add(1)

				go func() {
					defer moduleWg.Done()
					defer func() {
						// Catch panic in module execution
						if r := recover(); r != nil {
							moduleErr = fmt.Errorf("module %s panicked: %v", currentNode.instanceID, r)
							fmt.Fprintf(os.Stderr, "[FATAL] Panic in module '%s': %v\n", currentNode.instanceID, r)
						}
						close(outputChan)
					}()
					moduleErr = currentNode.module.Execute(execContext, inputsForNode, outputChan)
				}()

				for output := range outputChan {
					if output.FromModuleName == "" {
						output.FromModuleName = currentNode.instanceID
					}

					currentNode.outputs[output.DataKey] = output // Store in node's local outputs

					// dataCtxKey := fmt.Sprintf("%s.%s", currentNode.instanceID, output.DataKey)
					dataCtxKey := output.DataKey
					// Use a method that appends if key exists and is a list, otherwise creates a list.
					// This is what we had before and caused the CLI to expect []interface{}.
					// So, the CLI MUST handle this.
					o.dataCtx.AddOrAppendToList(dataCtxKey, output.Data) // Use AddOrAppendToList for module outputs

					if output.Error != nil {
						// fmt.Fprintf(os.Stderr, "[ERROR] Output error from module '%s' for DataKey '%s': %v\n", currentNode.instanceID, output.DataKey, output.Error)
						log.Error().Msgf("[ERROR] Output error from module '%s' for DataKey '%s': %v", currentNode.instanceID, output.DataKey, output.Error)
						// Decide if an output error should fail the module or just be logged.
						// For now, log it. Module's main error (moduleErr) will determine module status.
					} else {
						mlogger.Debug().
							Str("data_key", output.DataKey).Msgf("Output module")
					}
				}

				moduleWg.Wait() // Wait for the module's Execute goroutine (and panic recovery) to finish

				currentNode.endTime = time.Now()
				duration := currentNode.endTime.Sub(currentNode.startTime)

				completedMutex.Lock()
				if moduleErr != nil {
					currentNode.status = StatusFailed
					currentNode.err = moduleErr
					log.Err(moduleErr).Msgf("Module '%s' failed after %s: %v", currentNode.instanceID, duration, moduleErr)
					setOverallError(moduleErr)
				} else {
					currentNode.status = StatusCompleted
					mlogger.Info().Msgf("Module completed in %s.", duration)
				}
				executionCompleted[currentNode.instanceID] = true
				completedMutex.Unlock()
				nodeDoneSignal <- currentNode.instanceID // Signal completion
			}(node, nodeInputs)
		} // end for each node

		if !madeProgressInIteration && len(executionCompleted) < len(o.moduleNodes) {
			// If no new nodes could be started, but not all are done,
			// wait for a node to complete or context to be cancelled.
			select {
			case <-nodeDoneSignal:
				// A node finished, loop again to check for newly runnable nodes
			case <-ctx.Done():
				fmt.Println("[INFO] Orchestrator: Main context cancelled during execution.")
				setOverallError(ctx.Err()) // Set overall error to context error
			}
		}
		if overallError != nil && ctx.Err() == nil { // If a module error occurred and not due to context cancellation
			log.Warn().Err(overallError).Msgf("Orchestrator: Halting DAG due to module error: %v", overallError)
			// cancel all running/pending module contexts if possible (requires more complex context management per node)
			break // Exit main loop
		}
		if ctx.Err() != nil { // If main context was cancelled
			o.dataCtx.logger.Err(ctx.Err()).Msg("Orchestrator: Main context error, exiting DAG.")
			break
		}

	} // end while not all completed

	activeGoroutines.Wait() // Wait for any launched goroutines to finish

	oStatus := "success"
	if overallError != nil {
		oStatus = "failure"
	}

	o.logger.Info().
		Str("status", oStatus).Msg("DAG execution finished")

	return o.dataCtx.GetAll(), overallError
}
