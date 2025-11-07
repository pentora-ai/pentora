// pkg/engine/registry.go
// Package engine provides the core functionality for managing and executing modules.
package engine

import (
	"fmt"
)

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
func GetModuleInstance(instanceID, name string, config map[string]interface{}) (Module, error) {
	factory, ok := moduleRegistry[name]
	if !ok {
		return nil, fmt.Errorf("no module factory registered for name: %s", name)
	}
	moduleInstance := factory()
	if err := moduleInstance.Init(instanceID, config); err != nil {
		return nil, fmt.Errorf("failed to initialize module '%s': %w", name, err)
	}
	return moduleInstance, nil
}

// GetRegisteredModuleFactories returns a shallow copy of the module registry.
// This allows components like the DAGPlanner to discover available modules
// and access their factory functions to get metadata or create instances.
// Returning a copy prevents external modification of the original registry map.
func GetRegisteredModuleFactories() map[string]ModuleFactory {
	// logger := log.With().Str("component", "ModuleRegistry").Logger()
	// logger.Debug().Int("count", len(moduleRegistry)).Msg("Getting all registered module factories")

	// Create a copy to prevent external modification of the registry map itself.
	// The factories themselves are still references, but the map is new.
	registryCopy := make(map[string]ModuleFactory, len(moduleRegistry))
	for name, factory := range moduleRegistry {
		registryCopy[name] = factory
	}
	return registryCopy
}

// GetAllModuleMetadata creates temporary instances of all registered modules
// to retrieve their metadata. This can be resource-intensive if modules
// have heavy new() functions, so use judiciously or cache the results.
// The DAGPlanner would typically call this once during its initialization.
func GetAllModuleMetadata() ([]ModuleMetadata, error) {
	// logger := log.With().Str("component", "ModuleRegistry").Logger()
	// logger.Debug().Int("registry_size", len(moduleRegistry)).Msg("Getting metadata for all registered modules")

	allMetadata := make([]ModuleMetadata, 0, len(moduleRegistry))
	// if len(moduleRegistry) == 0 {
	// logger.Warn().Msg("No modules are registered in the moduleRegistry.")
	// It might be an error or just an early stage of initialization.
	// Depending on when this is called, returning an error might be appropriate.
	// For now, return empty list and no error if registry is empty.
	// }

	for name, factory := range moduleRegistry {
		// It's important that factory() creates a lightweight instance
		// primarily for metadata retrieval, without heavy initialization.
		// The actual configuration-based Init() is called by GetModuleInstance.
		moduleInstance := factory()
		if moduleInstance == nil {
			// logger.Error().Str("module_name", name).Msg("Module factory returned a nil instance")
			return nil, fmt.Errorf("module factory for '%s' returned a nil instance", name)
		}
		meta := moduleInstance.Metadata()
		// Ensure the metadata's Name field (module type name) is consistent with the registration name.
		if meta.Name == "" || meta.Name != name {
			// logger.Warn().Str("registered_name", name).Str("metadata_name", meta.Name).Msg("Module metadata Name mismatch or empty, using registered name.")
			meta.Name = name // Use the registered name as the canonical module type name
		}
		allMetadata = append(allMetadata, meta)
	}
	// logger.Info().Int("metadata_count", len(allMetadata)).Msg("Retrieved metadata for all modules")
	return allMetadata, nil
}
