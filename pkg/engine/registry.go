// pkg/engine/registry.go
// Package engine provides the core functionality for managing and executing modules.
package engine

import "fmt"

// ModuleFactory is a function that creates an instance of a module.
// This allows the orchestrator to dynamically load and instantiate modules.
type ModuleFactory func() Module // Config could be passed to factory or Init

// Global module registry
var moduleRegistry = make(map[string]ModuleFactory)

// RegisterModuleFactory adds a module factory to the registry.
// The `name` should correspond to the `module_type` used in DAG definitions.
func RegisterModuleFactory(name string, factory ModuleFactory) {
	if _, exists := moduleRegistry[name]; exists {
		// Handle duplicate registration, perhaps log a warning or error
		fmt.Printf("Warning: Module factory for '%s' is being overwritten.\n", name)
	}
	moduleRegistry[name] = factory
}

// GetModuleInstance creates a new instance of a module given its registered name
// and initializes it with the provided configuration.
func GetModuleInstance(name string, config map[string]interface{}) (Module, error) {
	factory, ok := moduleRegistry[name]
	if !ok {
		return nil, fmt.Errorf("no module factory registered for name: %s", name)
	}
	moduleInstance := factory()
	if err := moduleInstance.Init(config); err != nil {
		return nil, fmt.Errorf("failed to initialize module '%s': %w", name, err)
	}
	return moduleInstance, nil
}
